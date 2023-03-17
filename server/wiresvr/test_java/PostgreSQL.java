import java.sql.*;
import java.util.Properties;

public class PostgreSQL {
    public static void main(String[] args) throws ClassNotFoundException {
        Class.forName("org.postgresql.Driver");

        String connurl = "jdbc:postgresql://127.0.0.1:5651/postgres";
        Properties props = new Properties();
        // props.setProperty("user", "sys");
        // props.setProperty("password", "mamanger");
        props.setProperty("ssl", "false");
        
        // IMPORTENT!!!!
        // default is 'extended' which is not supported by wiresvr
        //
        props.setProperty("preferQueryMode", "simple"); 

        try (Connection conn = DriverManager.getConnection(connurl, props);) {
            //Statement stmt = conn.createStatement();
            //ResultSet rs = stmt.executeQuery("SELECT name, time, value FROM example WHERE name = 'wave.sin' ORDER BY time DESC LIMIT 10");
            
            String tag = "wave.sin";
            int limit = 10;

            PreparedStatement stmt = conn.prepareStatement("SELECT name, time, value FROM example WHERE name = ? ORDER BY time DESC LIMIT ?");
            stmt.setString(1, tag);
            stmt.setInt(2, limit);
            ResultSet rs = stmt.executeQuery();

            while (rs.next()) {
                String name = rs.getString("name");
                Timestamp ts = rs.getTimestamp("time");
                Double value = rs.getDouble("value");

                System.out.println(name +" "+ ts.toString()+" "+ value.toString());
            }
            rs.close();
            stmt.close();
            conn.close();
        } catch (SQLException e) {
            e.printStackTrace();
        }
    }
}