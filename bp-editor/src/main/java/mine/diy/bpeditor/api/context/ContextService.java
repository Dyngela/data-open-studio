package mine.diy.bpeditor.api.context;

import lombok.AccessLevel;
import lombok.RequiredArgsConstructor;
import lombok.experimental.FieldDefaults;
import lombok.extern.log4j.Log4j2;
import mine.diy.bpeditor.exceptions.ExceptionHandler;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
@RequiredArgsConstructor
@FieldDefaults(level = AccessLevel.PRIVATE, makeFinal = true)
@Log4j2
public class ContextService {
    ContextRepo contextRepo;

    public ContextEntity saveContext(ContextEntity contextEntity) {
        return contextRepo.save(contextEntity);
    }

    public ContextEntity getContextById(Integer id) {
        return contextRepo.findById(id).orElse(null);
    }

    public ContextEntity getContextByName(String name) {
        return contextRepo.findByName(name).orElseThrow(() -> new ExceptionHandler("Context not found"));
    }

    public List<ContextEntity> getAllContext() {
        return contextRepo.findAll();
    }

    public void deleteContext(Integer id) {
        contextRepo.deleteById(id);
    }
}
