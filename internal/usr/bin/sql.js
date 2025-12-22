const process = require('/lib/process');
const { parseArgs, splitFields } = require('/lib/util');
const { Client } = require('/usr/lib/machcli');

const options = {
    help: { type: 'boolean', short: 'h', description: 'Show this help message', default: false },
    host: { type: 'string', short: 'H', description: 'Database host', default: '127.0.0.1' },
    port: { type: 'number', short: 'P', description: 'Database port', default: 5656 },
    user: { type: 'string', description: 'Database user', default: 'sys' },
    password: { type: 'string', description: 'Database password', default: 'manager' },
    output: { type: 'string', short: 'o', description: "output file (default:'-' stdout)", default: '-' },
    format: { type: 'string', short: 'f', description: "output format (box, csv, json, ndjson)", default: 'box' },
    compress: { type: 'string', description: "compression type (none, gzip)", default: 'none' },
    delimiter: { type: 'string', short: 'd', description: "delimiter for csv format", default: ',' },
    rownum: { type: 'boolean', description: "show row numbers", default: true },
    timeformat: { type: 'string', short: 't', description: "time format [ns|ms|s|<timeformat>]", default: 'default' },
    tz: { type: 'string', description: "time zone for handling datetime (default: time zone)", default: 'local' },
    heading: { type: 'boolean', description: "print header", default: true },
    footer: { type: 'boolean', description: "print footer message", default: true },
    pause: { type: 'boolean', description: "pause for the screen paging", default: true },
    timing: { type: 'boolean', short: 'T', description: "print elapsed time", default: false },
    precision: { type: 'number', short: 'p', description: "set precision of float value to force round", default: -1 },
}
let showHelp = true;
let config = {};
let args = {};
try {
    const parsed = parseArgs(process.argv.slice(2), {
        options,
        allowPositionals: true,
        allowNegative: true,
        positionals: [
            { name: 'sql', type: 'string', variadic: true, description: 'SQL query to execute' }
        ]
    });
    config = parsed.values;
    args = parsed.namedPositionals;
    showHelp = config.help
}
catch (err) {
    console.println(err.message);
}

function showUsage(usage, options) {
    console.println("Usage:", usage);
    console.println("Options:");
    for (const [key, opt] of Object.entries(options)) {
        const short = opt.short ? `-${opt.short}, ` : '    ';
        console.println(`  ${short}--${key.padEnd(11, ' ')} ${opt.description} (default: ${opt.default})`);
    }
}

if (showHelp || (!args.sql) || args.sql.length === 0) {
    showUsage("sql [options] sql", options);
    process.exit(0);
}

const fields = splitFields(args.sql.join(' '));
if (fields[0].toLowerCase() === 'show') {
    console.println(`SHOW ${config.format} [${fields.slice(1).join(' / ')}] command recognized. (not yet implemented)`);
    process.exit(0);
}

const pretty = require('/usr/lib/pretty');

let db, conn, rows;
try {
    db = new Client(config);
    conn = db.connect();
    rows = conn.query(args.sql.join(' '));

    let box = pretty.Table({
        timeformat: config.timeformat,
        tz: config.tz,
        precision: config.precision,
    });
    if(config.rownum) box.setAutoIndex(true);
    box.appendHeader(rows.columnNames());

    // fetch and print rows
    for (const row of rows) {
        // spread row values
        box.appendRow(box.row(...row));
    }
    // render box
    console.println(box.render());
    // print footer message
    console.println(rows.message);
} catch (err) {
    console.println("Error: ", err.message);
} finally {
    rows && rows.close();
    conn && conn.close();
    db && db.close();
}