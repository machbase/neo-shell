import java.sql.*;
import java.util.Properties;

public class PostgreSQL {
    public static void main(String[] args) throws ClassNotFoundException {
        Class.forName("org.postgresql.Driver");

        String connurl = "jdbc:postgresql://127.0.0.1:5651/postgres";
        //String connurl = "jdbc:postgresql://127.0.0.1:5432/postgres";
        Properties props = new Properties();
        // props.setProperty("user", "sys");
        // props.setProperty("password", "mamanger");
        props.setProperty("ssl", "false");

        try (Connection conn = DriverManager.getConnection(connurl, props);) {
            Statement stmt = conn.createStatement();
            // ResultSet rs = stmt.executeQuery ("SELECT * from AAA");
            ResultSet rs = stmt.executeQuery("SELECT name, time, value FROM example WHERE name = 'wave.sin' LIMIT 10");

            while (rs.next()) {
                String name = rs.getString("name");

                System.out.println(name);
            }
            rs.close();
            stmt.close();
            conn.close();
        } catch (SQLException e) {
            e.printStackTrace();
        }
    }
}