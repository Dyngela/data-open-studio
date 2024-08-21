package mine.diy.bpeditor.api.context;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.Optional;

@Repository
public interface ContextRepo extends JpaRepository<ContextEntity, Integer> {
    Optional<ContextEntity> findByName(String name);
}
