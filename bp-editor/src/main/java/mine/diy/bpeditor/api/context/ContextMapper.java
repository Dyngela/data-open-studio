package mine.diy.bpeditor.api.context;

import mine.diy.bpeditor.api.context.request.CreateContext;
import mine.diy.bpeditor.api.context.request.UpdateContext;
import mine.diy.bpeditor.api.context.response.FindContext;
import org.mapstruct.Mapper;
import org.mapstruct.MappingConstants;
import org.mapstruct.ReportingPolicy;

import java.util.List;

@Mapper(unmappedSourcePolicy = ReportingPolicy.IGNORE, unmappedTargetPolicy = ReportingPolicy.IGNORE,
        typeConversionPolicy = ReportingPolicy.ERROR, componentModel = MappingConstants.ComponentModel.SPRING)
public interface ContextMapper {
    FindContext toFindContext(ContextEntity contextEntity);
    List<FindContext> toFindContext(List<ContextEntity> contextEntities);

    ContextEntity toEntity(CreateContext createContext);
    ContextEntity toEntity(UpdateContext updateContext);
}
