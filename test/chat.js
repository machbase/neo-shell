'use strict';

const http = require('http');
const { WebSocket } = require('ws');
const { ReadLine } = require('readline');

// events: "answer-start", "answer-stop"
class Chat extends EventEmitter {
    constructor(options = {}) {
        super();

        this.options = {
            protocol: 'http:',
            host: '127.0.0.1',
            port: 5654,
            user: 'sys',
            password: 'manager'
        };
        this.options = { ...this.options, ...options }
    }
    login() {
        console.println(`Logging in to ${this.options.host}:${this.options.port} as ${this.options.user}...`);
        return new Promise((resolve, reject) => {
            const req = http.request({
                method: 'POST',
                protocol: this.options.protocol,
                host: this.options.host,
                port: this.options.port,
                path: '/web/api/login',
                headers: {
                    'Content-Type': 'application/json'
                }
            });
            req.on('response', (res) => {
                const result = res.json();
                if (!result.success) {
                    reject(new Error('Login failed: ' + result.reason));
                    return;
                }
                this.accessToken = result.accessToken;
                this.refreshToken = result.refreshToken;
                const wsUrl = `${this.options.protocol === 'https:' ? 'wss:' : 'ws:'}//${this.options.host}:${this.options.port}/web/api/console/1234/data?token=${this.accessToken}`;
                resolve(wsUrl);
            });
            req.on('error', (err) => {
                reject(err);
            });
            const body = JSON.stringify({
                loginName: this.options.user,
                password: this.options.password
            });
            req.write(body);
            req.end();
        });
    }
    answering(reply) {
        // reply: {data:{"type":"msg","msg":{"body":null,"id":1234,"type":"answer-start","ver":"1.0"}}
        const obj = JSON.parse(reply.data);
        switch (obj.type) {
            case 'msg':
                switch (obj.msg.type) {
                    case 'answer-start':
                        this.endAnswer = false;
                    case 'stream-message-start':
                    case 'stream-block-start':
                    case 'stream-block-delta':
                    case 'stream-block-stop':
                    case 'stream-message-stop':
                        if (obj.msg.body && obj.msg.body.data && obj.msg.body.data.length > 0) {
                            let text = obj.msg.body.data;
                            console.print(text)
                            this.emit('answer-start', text);
                        }
                        break;
                    case 'answer-stop':
                        this.endAnswer = true;
                        this.emit('answer-stop', obj.msg.body);
                        break;
                    default:
                        console.println('Unknown msg type:', obj.msg.type);
                        return;
                }
                break;
            default:
                console.println('Unknown data type:', reply.data.trim());
                return;
        }
    }
}

function question(chat, ws, msg) {
    chat.endAnswer = false;
    const quest = JSON.stringify({
        type: 'msg',
        msg: {
            ver: '1.0',
            id: 1234,
            type: 'question',
            body: {
                provider: 'claude',
                model: 'claude-haiku-4-5-20251001',
                text: msg
            }
        },
    });
    try {
        ws.send(quest);
    } catch (e) {
        console.error("Failed to send question:", e.message);
        return;
    }

    const handle = (ws) => {
        if (!chat.endAnswer) {
            setImmediate(() => {
                handle(ws);
            });
            return;
        }
        console.println("\n"); // Answer complete.
        setImmediate(() => {
            loop(chat, ws);
        });
    }
    setImmediate(() => {
        handle(ws);
    });
}

const r = new ReadLine({
    prompt: (lineno) => {
        return lineno == 0 ? `chat > ` : `.... > `;
    },
});

function loop(chat, ws) {
    if (!chat.connected) {
        console.println('WebSocket is not connected.');
        return;
    }
    const line = r.readLine();
    if (line instanceof Error) {
        throw line;
    }
    if (line === null || line.toLowerCase() === 'exit') {
        console.println('Exiting...');
        ws.close();
        return;
    }

    if (line.trim() === '') {
        setImmediate(() => {
            loop(chat, ws);
        });
    } else {
        question(chat, ws, line);
    }
}

const chat = new Chat({ host: '192.168.1.165', port: 5654 });
chat.login()
    .then((wsUrl) => {
        const ws = new WebSocket(wsUrl);
        ws.on('open', () => {
            setImmediate(() => {
                chat.connected = true;
                console.println('You can type your messages now. Type "exit" to quit.');
                loop(chat, ws);
            });
        });
        ws.on('error', (err) => {
            console.println(err.message);
        });
        ws.on('close', () => {
            chat.connected = false;
        });
        ws.on('message', (data) => {
            chat.answering(data);
        });
    })
    .catch((err) => {
        console.error('Login failed:', err.message);
    });