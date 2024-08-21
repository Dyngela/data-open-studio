package mine.diy.bpeditor;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.transaction.annotation.EnableTransactionManagement;

@SpringBootApplication
public class BpEditorApplication {

    public static void main(String[] args) {
        SpringApplication.run(BpEditorApplication.class, args);
    }

}
