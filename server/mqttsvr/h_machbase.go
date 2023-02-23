package mqttsvr

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/machbase/neo-shell/codec"
	"github.com/machbase/neo-shell/server/mqttsvr/mqtt"
	"github.com/machbase/neo-shell/server/msg"
	"github.com/machbase/neo-shell/stream"
	spi "github.com/machbase/neo-spi"
	"github.com/tidwall/gjson"
)

func (svr *Server) onMachbase(evt *mqtt.EvtMessage, prefix string) error {
	tick := time.Now()
	topic := evt.Topic
	topic = strings.TrimPrefix(topic, prefix+"/")
	peer, ok := svr.mqttd.GetPeer(evt.PeerId)
	peerLog := peer.GetLog()

	reply := func(msg any) {
		if ok {
			buff, err := json.Marshal(msg)
			if err != nil {
				return
			}
			peer.Publish(prefix+"/reply", 1, buff)
		}
	}
	if topic == "query" {
		////////////////////////
		// query
		req := &msg.QueryRequest{}
		rsp := &msg.QueryResponse{Reason: "not specified"}
		err := json.Unmarshal(evt.Raw, req)
		if err != nil {
			rsp.Reason = err.Error()
			rsp.Elapse = time.Since(tick).String()
			reply(rsp)
			return nil
		}
		Query(svr.db, req, rsp)
		rsp.Elapse = time.Since(tick).String()
		reply(rsp)
	} else if strings.HasPrefix(topic, "write") {
		////////////////////////
		// write
		req := &msg.WriteRequest{}
		rsp := &msg.WriteResponse{Reason: "not specified"}
		err := json.Unmarshal(evt.Raw, req)
		if err != nil {
			rsp.Reason = err.Error()
			rsp.Elapse = time.Since(tick).String()
			reply(rsp)
			return nil
		}
		if len(req.Table) == 0 {
			req.Table = strings.TrimPrefix(topic, "write/")
		}

		if len(req.Table) == 0 {
			rsp.Reason = "table is not specified"
			rsp.Elapse = time.Since(tick).String()
			reply(rsp)
			return nil
		}
		Write(svr.db, req, rsp)
		rsp.Elapse = time.Since(tick).String()
		reply(rsp)
	} else if strings.HasPrefix(topic, "append/") {
		////////////////////////
		// append
		table := strings.ToUpper(strings.TrimPrefix(topic, "append/"))
		if len(table) == 0 {
			return nil
		}

		toks := strings.Split(table, ":")
		var compress = ""
		var format = "json"
		for i, t := range toks {
			if i == 0 {
				table = t
			} else {
				t = strings.ToLower(t)
				switch t {
				case "gzip":
					compress = t
				case "csv":
					format = t
				case "json":
					format = t
				}
			}
		}

		var err error
		var appenderSet []spi.Appender
		var appender spi.Appender

		val, exists := svr.appenders.Get(evt.PeerId)
		if exists {
			appenderSet = val.([]spi.Appender)
			for _, a := range appenderSet {
				if a.TableName() == table {
					appender = a
					break
				}
			}
		}
		if appender == nil {
			appender, err = svr.db.Appender(table)
			if err != nil {
				svr.log.Errorf("fail to create appender, %s", err.Error())
				return nil
			}
			if len(appenderSet) == 0 {
				appenderSet = []spi.Appender{}
			}
			appenderSet = append(appenderSet, appender)
			svr.appenders.Set(evt.PeerId, appenderSet)
		}

		payload := evt.Raw

		if compress == "gzip" {
			gr, err := gzip.NewReader(bytes.NewBuffer(payload))
			defer func() {
				if gr == nil {
					return
				}
				err = gr.Close()
				if err != nil {
					peerLog.Errorf("fail to close decompressor, %s", err.Error())
				}
			}()
			if err != nil {
				peerLog.Errorf("fail to gunzip, %s", err.Error())
				return nil
			}
			payload, err = io.ReadAll(gr)
			if err != nil {
				peerLog.Errorf("fail to gunzip, %s", err.Error())
				return nil
			}
		}

		if format == "json" {
			result := gjson.ParseBytes(payload)
			head := result.Get("0")
			if head.IsArray() {
				// if payload contains multiple tuples
				cols, err := appender.Columns()
				if err != nil {
					peerLog.Errorf("fail to get appender columns, %s", err.Error())
					return nil
				}
				result.ForEach(func(key, value gjson.Result) bool {
					fields := value.Array()
					vals, err := convAppendColumns(fields, cols, appender.TableType())
					if err != nil {
						return false
					}
					err = appender.Append(vals...)
					if err != nil {
						peerLog.Warnf("append fail %s %d %s [%+v]", table, appender.TableType(), err.Error(), vals)
						return false
					}
					return true
				})
				return err
			} else {
				// a single tuple
				fields := result.Array()
				cols, err := appender.Columns()
				if err != nil {
					peerLog.Errorf("fail to get appender columns, %s", err.Error())
					return nil
				}
				vals, err := convAppendColumns(fields, cols, appender.TableType())
				if err != nil {
					return err
				}
				err = appender.Append(vals...)
				if err != nil {
					peerLog.Warnf("append fail %s %d %s [%+v]", table, appender.TableType(), err.Error(), vals)
					return err
				}
				return nil
			}
		} else if format == "csv" {
			cols, _ := appender.Columns()
			decoder := codec.NewDecoderBuilder(format).
				SetInputStream(&stream.ReaderInputStream{Reader: bytes.NewReader(payload)}).
				SetColumns(cols).
				SetTimeFormat("ns").
				SetTimeLocation(time.UTC).
				SetCsvDelimieter(",").
				SetCsvHeading(false).
				Build()
			lineno := 0
			for {
				vals, err := decoder.NextRow()
				if err != nil {
					if err != io.EOF {
						peerLog.Errorf("append csv, %s", err.Error())
						return nil
					}
					break
				}
				err = appender.Append(vals...)
				if err != nil {
					peerLog.Errorf("append csv, %s", err.Error())
					break
				}
				lineno++
			}
			peerLog.Infof("%s appended %d record(s)", evt.Topic, lineno)
		}
	}
	return nil
}

func convAppendColumns(fields []gjson.Result, cols spi.Columns, tableType spi.TableType) ([]any, error) {
	fieldsOffset := 0
	colsNum := len(cols)
	vals := []any{}
	switch tableType {
	case spi.LogTableType:
		if colsNum == len(fields) {
			// num of columns is matched
		} else if colsNum+1 == len(fields) {
			vals = append(vals, fields[0].Int()) // timestamp included
			fieldsOffset = 1
		} else {
			return nil, fmt.Errorf("append fail, received fields not matched columns(%d)", colsNum)
		}

	default:
		if colsNum == len(fields) {
			// num of columns is matched
		} else {
			return nil, fmt.Errorf("append fail, received fields not matched columns(%d)", colsNum)
		}
	}

	for i, v := range fields[fieldsOffset:] {
		switch cols[i].Type {
		case spi.ColumnBufferTypeInt16:
			vals = append(vals, v.Int())
		case spi.ColumnBufferTypeInt32:
			vals = append(vals, v.Int())
		case spi.ColumnBufferTypeInt64:
			vals = append(vals, v.Int())
		case spi.ColumnBufferTypeString:
			vals = append(vals, v.Str)
		case spi.ColumnBufferTypeDatetime:
			vals = append(vals, v.Int())
		case spi.ColumnBufferTypeFloat:
			vals = append(vals, v.Float())
		case spi.ColumnBufferTypeDouble:
			vals = append(vals, v.Float())
		case spi.ColumnBufferTypeIPv4:
			vals = append(vals, v.Str)
		case spi.ColumnBufferTypeIPv6:
			vals = append(vals, v.Str)
		case spi.ColumnBufferTypeBinary:
			return nil, errors.New("append fail, binary column is not supproted via JSON payload")
		}
	}

	return vals, nil
}
