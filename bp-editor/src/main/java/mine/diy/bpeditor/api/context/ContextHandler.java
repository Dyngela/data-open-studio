package mine.diy.bpeditor.api.context;

import jakarta.validation.Valid;
import lombok.AccessLevel;
import lombok.RequiredArgsConstructor;
import lombok.experimental.FieldDefaults;
import lombok.extern.slf4j.Slf4j;
import mine.diy.bpeditor.api.context.request.CreateContext;
import mine.diy.bpeditor.api.context.request.UpdateContext;
import mine.diy.bpeditor.api.context.response.FindContext;
import mine.diy.bpeditor.exceptions.ExceptionHandler;
import org.hibernate.sql.Update;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RequiredArgsConstructor
@Slf4j
@FieldDefaults(level = AccessLevel.PRIVATE, makeFinal = true)
@RestController
@RequestMapping("api/v1/context")
public class ContextHandler {
    ContextService contextService;
    ContextMapper contextMapper;

    @GetMapping("/{id}")
    public FindContext findContext(@PathVariable Integer id) {
        return contextMapper.toFindContext(contextService.getContextById(id));
    }

    @GetMapping("")
    public List<FindContext> findAllContext() {
        return contextMapper.toFindContext(contextService.getAllContext());
    }

    @PostMapping("")
    public FindContext createContext(@Valid @RequestBody CreateContext createContext) {
        return contextMapper.toFindContext(contextService.saveContext(contextMapper.toEntity(createContext)));
    }

    @PutMapping()
    public FindContext updateContext(@Valid @RequestBody UpdateContext updateContext) {
        return contextMapper.toFindContext(contextService.saveContext(contextMapper.toEntity(updateContext)));
    }

    @DeleteMapping("/{id}")
    public void deleteContext(@PathVariable Integer id) {
        contextService.deleteContext(id);
    }
}
