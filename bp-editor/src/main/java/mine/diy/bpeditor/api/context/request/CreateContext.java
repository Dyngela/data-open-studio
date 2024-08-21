package mine.diy.bpeditor.api.context.request;

import lombok.*;

@Setter
@Getter
@AllArgsConstructor
@NoArgsConstructor
@ToString
public class CreateContext {
    String name;
    String host;
    String port;
    String username;
    String password;
    String database;
}
