const { Client } = require("machcli");
const process = require('process');
const args = process.argv.slice(2);
try {
    db = new Client({ host: '192.168.1.165' });
    conn = db.connect();
    rows = conn.query("SELECT * from TAG order by time limit ?", args[0] ?? 1);
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