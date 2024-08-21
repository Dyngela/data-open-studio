package mine.diy.bpeditor.api.context.response;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
@AllArgsConstructor
public class FindContext {
    Integer id;
    String name;
    String host;
    String port;
    String username;
    String password;
    String database;
}
