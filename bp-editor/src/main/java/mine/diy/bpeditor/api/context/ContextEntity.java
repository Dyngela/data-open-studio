package mine.diy.bpeditor.api.context;

import jakarta.persistence.*;
import lombok.*;
import lombok.experimental.FieldDefaults;

@Entity
@Setter
@Getter
@ToString
@NoArgsConstructor
@AllArgsConstructor
@Table(name = "context")
@FieldDefaults(level = AccessLevel.PRIVATE)
public class ContextEntity {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    @Column(name = "id", unique = true, nullable = false)
    Integer id;

    @Column(name = "name", unique = true)
    String name;

    @Column(name = "host")
    String host;

    @Column(name = "port")
    String port;

    @Column(name = "username")
    String username;

    @Column(name = "password")
    String password;

    @Column(name = "database")
    String database;
}
