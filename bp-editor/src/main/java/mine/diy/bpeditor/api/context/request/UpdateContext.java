package mine.diy.bpeditor.api.context.request;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
@AllArgsConstructor
public class UpdateContext {
    Integer id;
    String name;
    String host;
    String port;
    String username;
    String password;
    String database;
}
