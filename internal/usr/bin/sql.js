'use strict';

const process = require('process');
const parseArgs = require('util/parseArgs');
const { Client } = require('/usr/lib/machcli');
const pretty = require('/usr/lib/pretty');

const options = {
    help: { type: 'boolean', short: 'h', description: 'Show this help message', default: false },
    output: { type: 'string', short: 'o', description: "output file (default:'-' stdout)", default: '-' },
    compress: { type: 'string', description: "compression type (none, gzip)", default: 'none' },
    timing: { type: 'boolean', short: 'T', description: "print elapsed time", default: false },
    showTz: { type: 'boolean', short: 'Z', description: "show time zone in datetime column header", default: false },
    ...pretty.TableArgOptions,
}
const positionals = [
    { name: 'sql', type: 'string', variadic: true, description: 'SQL query to explain' }
];

let showHelp = true;
let config = {};
let args = {};
try {
    const parsed = parseArgs(process.argv.slice(2), {
        options,
        allowPositionals: true,
        allowNegative: true,
        positionals: positionals
    });
    config = parsed.values;
    args = parsed.namedPositionals;
    showHelp = config.help
}
catch (err) {
    console.println(err.message);
}

if (showHelp || (!args.sql) || args.sql.length === 0) {
    console.println(parseArgs.formatHelp({
        usage: 'Usage: sql [options] <sql>',
        options,
        positionals: positionals
    }));
    process.exit(showHelp ? 0 : 1);
}

const sqlText = args.sql.join(' ');
let db, conn, rows;
try {
    db = new Client(config);
    conn = db.connect();
    rows = conn.query(sqlText);

    let tick = process.now();
    let box = pretty.Table(config);
    if (config.showTz) {
        let columnLabels = [];
        for (let i = 0; i < rows.columnTypes.length; i++) {
            if (rows.columnTypes[i] == 'datetime') {
                columnLabels.push(rows.columnNames[i] + `(${config.tz})`);
            } else {
                columnLabels.push(rows.columnNames[i])
            }
        }
        box.appendHeader(columnLabels);
    } else {
        box.appendHeader(rows.columnNames);
    }
    box.setColumnTypes(rows.columnTypes);

    for (const row of rows) {
        // spread row values
        box.append([...row]);
        if (box.requirePageRender()) {
            // render page
            console.println(box.render());
            // wait for user input to continue if pause is enabled
            if (!box.pauseAndWait()) {
                break;
            }
        }
    }
    // set footer message
    box.setCaption(rows.message)
    // render remaining rows
    console.println(box.close());
    // print elapsed time
    if (config.timing) {
        console.println(`Elapsed time: ${pretty.Durations(process.now().unixNano() - tick.unixNano())}`);
    }
} catch (err) {
    console.println("Error: ", err.message);
} finally {
    rows && rows.close();
    conn && conn.close();
    db && db.close();
}
