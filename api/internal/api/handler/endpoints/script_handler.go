package endpoints

import (
	"api"
	"api/internal/api/handler/middleware"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type CodeRequest struct {
	Code string `json:"code"`
}

type CodeResponse struct {
	Output           string                  `json:"output"`
	Error            bool                    `json:"error"`
	CompilationError *CompilationErrorDetail `json:"compilationError,omitempty"`
}

type CompilationErrorDetail struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

type scriptHandler struct {
	logger zerolog.Logger
	config api.AppConfig
}

var errScriptCompilationFailed = errors.New("script compilation failed")

var compilationErrorInlinePattern = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*Error):\s*(.+?)\s*\(<user-script>, line (\d+)\)$`)
var compilationErrorTracePattern = regexp.MustCompile(`File "<user-script>", line (\d+)`)
var compilationErrorTypePattern = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*Error):\s*(.+)$`)

// pythonValidatorScript checks syntax via ast.parse (same quality as py_compile but with
// better column info) and also verifies that a top-level "transform" function is defined.
// pythonValidatorScript performs three checks (stdlib only, no extra packages):
//  1. Syntax   – ast.parse with precise line/column info
//  2. Structure – transform function must be defined at module level
//  3. Scope    – detects NameError (undefined names) inside every function body
const pythonValidatorScript = `import ast, sys, builtins as _b

with open(sys.argv[1], encoding='utf-8') as _f:
	_src = _f.read()
_src_lines = _src.splitlines()

# -- 1. Syntax ----------------------------------------------------------------
try:
	_tree = ast.parse(_src, filename='<user-script>')
except SyntaxError as _e:
	_out = ['  File "<user-script>", line {}'.format(_e.lineno or 0)]
	if _e.text:
		_out.append('    {}'.format(_e.text.rstrip()))
		_out.append('    {}^'.format(' ' * max(0, (_e.offset or 1) - 1)))
	_out.append('{}: {}'.format(type(_e).__name__, _e.msg))
	print('\n'.join(_out))
	sys.exit(1)

# -- 2. transform must exist at module level ----------------------------------
if not any(isinstance(_n, ast.FunctionDef) and _n.name == 'transform' for _n in _tree.body):
	print('NameError: This script must define a top-level function named "transform"')
	sys.exit(1)

# -- 3. Scope-aware NameError detection --------------------------------------
_BUILTINS = frozenset(dir(_b))

def _extract_bound(t):
	"""Yield every name bound by an assignment target (handles tuple unpacking)."""
	if isinstance(t, ast.Name):
		yield t.id
	elif isinstance(t, (ast.Tuple, ast.List)):
		for e in t.elts:
			yield from _extract_bound(e)
	elif isinstance(t, ast.Starred):
		yield from _extract_bound(t.value)

def _mod_scope(tree):
	"""Collect all names visible at module scope."""
	s = set(_BUILTINS)
	for n in tree.body:
		if isinstance(n, ast.Import):
			for a in n.names:
				s.add(a.asname or a.name.split('.')[0])
		elif isinstance(n, ast.ImportFrom):
			for a in n.names:
				if a.name != '*':
					s.add(a.asname or a.name)
		elif isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
			s.add(n.name)
		elif isinstance(n, ast.Assign):
			for t in n.targets:
				for nm in _extract_bound(t):
					s.add(nm)
		elif isinstance(n, ast.AnnAssign):
			if isinstance(n.target, ast.Name) and n.value:
				s.add(n.target.id)
	return s

def _func_bound(func):
	"""Collect all names bound inside a function (params + local bindings).
	Does NOT descend into nested function/class bodies."""
	b = set()
	for arg in func.args.args + func.args.posonlyargs + func.args.kwonlyargs:
		b.add(arg.arg)
	if func.args.vararg:
		b.add(func.args.vararg.arg)
	if func.args.kwarg:
		b.add(func.args.kwarg.arg)
	q = list(func.body)
	while q:
		n = q.pop()
		if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef)):
			b.add(n.name)
			continue  # do not descend into nested func bodies
		if isinstance(n, ast.ClassDef):
			b.add(n.name)
			continue  # do not descend into nested class bodies
		if isinstance(n, (ast.Assign, ast.AugAssign)):
			for t in (n.targets if isinstance(n, ast.Assign) else [n.target]):
				for nm in _extract_bound(t):
					b.add(nm)
		elif isinstance(n, ast.AnnAssign):
			for nm in _extract_bound(n.target):
				b.add(nm)
		elif isinstance(n, (ast.For, ast.AsyncFor)):
			for nm in _extract_bound(n.target):
				b.add(nm)
		elif isinstance(n, ast.With):
			for item in n.items:
				if item.optional_vars:
					for nm in _extract_bound(item.optional_vars):
						b.add(nm)
		elif isinstance(n, ast.ExceptHandler):
			if n.name:
				b.add(n.name)
		elif hasattr(ast, 'NamedExpr') and isinstance(n, ast.NamedExpr):
			for nm in _extract_bound(n.target):
				b.add(nm)
		q.extend(ast.iter_child_nodes(n))
	return b

def _check(func, outer):
	"""Report the first undefined Name(Load) found in the function body."""
	scope = outer | _func_bound(func)
	q = list(func.body)
	while q:
		n = q.pop()
		# Skip nested scopes entirely – they resolve names in their own context
		if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
			continue
		if isinstance(n, ast.Name) and isinstance(n.ctx, ast.Load):
			if n.id not in scope and not n.id.startswith('__'):
				ln = n.lineno
				lt = _src_lines[ln - 1] if ln <= len(_src_lines) else ''
				print('  File "<user-script>", line {}'.format(ln))
				print('    ' + lt)
				print('    {}^'.format(' ' * n.col_offset))
				print("NameError: name '{}' is not defined".format(n.id))
				sys.exit(1)
		q.extend(ast.iter_child_nodes(n))

_gscope = _mod_scope(_tree)
for _n in _tree.body:
	if isinstance(_n, (ast.FunctionDef, ast.AsyncFunctionDef)):
		_check(_n, _gscope)
`

func newScriptHandler() *scriptHandler {
	return &scriptHandler{
		logger: api.Logger,
		config: api.GetConfig(),
	}
}

func ScriptHandler(router *graceful.Graceful) {
	h := newScriptHandler()

	routes := router.Group("/api/v1/script")
	routes.Use(middleware.AuthMiddleware(h.config))
	{
		routes.POST("/compile", h.compilationCheck)
	}
}

func (sh *scriptHandler) compilationCheck(c *gin.Context) {
	var req CodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	output, err := compilePythonCode(req.Code)
	if err != nil {
		compilationError := parseCompilationError(output)

		if errors.Is(err, errScriptCompilationFailed) {
			sh.logger.Warn().Str("compiler_output", output).Msg("Compilation failed")
		} else {
			sh.logger.Error().Err(err).Str("compiler_output", output).Msg("Compilation error")
		}

		c.JSON(http.StatusOK, CodeResponse{
			Output:           output,
			Error:            true,
			CompilationError: compilationError,
		})
		return
	}

	c.JSON(http.StatusOK, CodeResponse{
		Output: "Code compiled successfully",
		Error:  false,
	})
}

func compilePythonCode(code string) (string, error) {
	// ── Write user script ─────────────────────────────────────────────────────
	userFile, err := os.CreateTemp("", "user-script-*.py")
	if err != nil {
		return "Failed to prepare temporary file for script compilation.", fmt.Errorf("create temp file: %w", err)
	}
	userPath := userFile.Name()
	if err := userFile.Close(); err != nil {
		return "Failed to prepare temporary file for script compilation.", fmt.Errorf("close temp file: %w", err)
	}
	defer os.Remove(userPath)

	if err := os.WriteFile(userPath, []byte(code), 0o600); err != nil {
		return "Failed to write temporary file for script compilation.", fmt.Errorf("write temp file: %w", err)
	}

	// ── Write validator script ────────────────────────────────────────────────
	validatorFile, err := os.CreateTemp("", "validator-*.py")
	if err != nil {
		return "Failed to prepare validator for script compilation.", fmt.Errorf("create validator file: %w", err)
	}
	validatorPath := validatorFile.Name()
	if err := validatorFile.Close(); err != nil {
		return "Failed to prepare validator for script compilation.", fmt.Errorf("close validator file: %w", err)
	}
	defer os.Remove(validatorPath)

	if err := os.WriteFile(validatorPath, []byte(pythonValidatorScript), 0o600); err != nil {
		return "Failed to write validator for script compilation.", fmt.Errorf("write validator: %w", err)
	}

	// ── Try each Python entry-point ───────────────────────────────────────────
	pythonCommands := []struct {
		executable string
		args       []string
	}{
		{executable: "python", args: []string{validatorPath, userPath}},
		{executable: "python3", args: []string{validatorPath, userPath}},
		{executable: "py", args: []string{"-3", validatorPath, userPath}},
	}

	var lastErr error
	var lastOutput string

	for _, pythonCommand := range pythonCommands {
		cmd := exec.Command(pythonCommand.executable, pythonCommand.args...)

		out, err := cmd.CombinedOutput()
		if err == nil {
			return "", nil
		}

		if isCommandNotFound(err) {
			continue
		}

		compileOutput := normalizeCompilerOutput(string(out), userPath, validatorPath)
		if compileOutput == "" {
			// Some Windows launchers can return a failing exit code without stderr.
			// Keep trying other Python entry-points before returning a generic error.
			lastErr = fmt.Errorf("%s: %w", pythonCommand.executable, err)
			lastOutput = err.Error()
			continue
		}

		return compileOutput, fmt.Errorf("%w using %s: %w", errScriptCompilationFailed, pythonCommand.executable, err)
	}

	if lastErr != nil {
		if strings.TrimSpace(lastOutput) == "" || strings.EqualFold(strings.TrimSpace(lastOutput), "exit status 1") {
			return "Unable to run a working Python interpreter for compilation. Check python/python3/py installation.", lastErr
		}

		return lastOutput, lastErr
	}

	return "Python runtime not found. Install python3 or python to enable script compilation checks.", errors.New("python runtime not found")
}

func normalizeCompilerOutput(output string, tempPaths ...string) string {
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		return ""
	}

	normalizedOutput := trimmedOutput
	for _, p := range tempPaths {
		normalizedOutput = strings.ReplaceAll(normalizedOutput, p, "<user-script>")
		normalizedOutput = strings.ReplaceAll(normalizedOutput, filepath.Base(p), "<user-script>")
	}

	return strings.TrimPrefix(normalizedOutput, "Sorry: ")
}

func parseCompilationError(output string) *CompilationErrorDetail {
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		return nil
	}

	if matches := compilationErrorInlinePattern.FindStringSubmatch(trimmedOutput); matches != nil {
		line, _ := strconv.Atoi(matches[3])
		return &CompilationErrorDetail{
			Type:    matches[1],
			Message: matches[2],
			Line:    line,
		}
	}

	compilationError := &CompilationErrorDetail{Message: trimmedOutput}
	lines := strings.Split(trimmedOutput, "\n")

	if matches := compilationErrorTracePattern.FindStringSubmatch(trimmedOutput); matches != nil {
		line, _ := strconv.Atoi(matches[1])
		compilationError.Line = line
	}

	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			continue
		}

		if matches := compilationErrorTypePattern.FindStringSubmatch(line); matches != nil {
			compilationError.Type = matches[1]
			compilationError.Message = matches[2]
			break
		}
	}

	for _, line := range lines {
		if !strings.Contains(line, "^") {
			continue
		}

		if strings.Trim(strings.ReplaceAll(line, "^", ""), " \t") != "" {
			continue
		}

		compilationError.Column = strings.Index(line, "^") + 1
		break
	}

	return compilationError
}

func isCommandNotFound(err error) bool {
	var execErr *exec.Error
	return errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound)
}
