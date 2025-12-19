const process = require('process');
const { parseArgs } = require('/lib/util');
const { Client } = require("machcli");

const options = {
    help: { type: 'boolean', short: 'h', description: 'Show this help message' },
    host: { type: 'string', short: 'H', description: 'Database host', default: '127.0.0.1' },
    port: { type: 'number', short: 'P', description: 'Database port', default: 5656 },
    user: { type: 'string', short: 'u', description: 'Database user', default: 'sys' },
    password: { type: 'string', short: 'p', description: 'Database password', default: 'manager' },
}

const parsed = parseArgs(process.argv.slice(2), { options, allowPositionals: true });

const config = parsed.values;
const args = parsed.positionals;

if (config.help) {
    console.println("Usage: node query.js [options] [limit]");
    console.println("Options:");
    for (const [key, opt] of Object.entries(options)) {
        const short = opt.short ? `-${opt.short}, ` : '';
        console.println(`  ${short}--${key}: ${opt.description} (default: ${opt.default})`);
    }
    process.exit(0);
}

let db, conn, rows;
try {
    db = new Client(config);
    conn = db.connect();
    rows = conn.query("SELECT * from TAG order by time limit ?", (args && args[0]) ? args[0] : 1);
    for (const row of rows) {
        console.println("ROWNUM:", row._ROWNUM, "NAME:", row.NAME, "TIME:", row.TIME, "VALUE:", row.VALUE);
    }
    console.println(rows.message);
} catch (err) {
    console.println("Error: ", err.message);
} finally {
    rows && rows.close();
    conn && conn.close();
    db && db.close();
}