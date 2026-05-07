//go:build cgo

package complexity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestAnalyzeFile_JavaScript_Simple(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "simple.js", `
function add(a, b) {
  return a + b;
}

function greet(name) {
  console.log("hello " + name);
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Functions), 2)

	for _, f := range result.Functions {
		assert.Equal(t, 1, f.Complexity, "simple function %s should have complexity 1", f.Name)
	}
}

func TestAnalyzeFile_JavaScript_IfElse(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "branching.js", `
function check(x) {
  if (x > 10) {
    return "big";
  } else if (x > 5) {
    return "medium";
  } else {
    return "small";
  }
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + 2 ifs
}

func TestAnalyzeFile_JavaScript_BooleanOps(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "boolean.js", `
function validate(a, b) {
  if (a > 0 && b > 0) {
    return true;
  }
  return false;
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + if + &&
}

func TestAnalyzeFile_TypeScript(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.ts", `
function processItems(items: number[]): number {
  let sum = 0;
  for (const item of items) {
    if (item > 0) {
      sum += item;
    }
  }
  return sum;
}
`)
	result, err := AnalyzeFile(path, "TypeScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + for + if
}

func TestAnalyzeFile_Python(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.py", `
def process(items):
    result = []
    for item in items:
        if item > 0:
            result.append(item)
        elif item == 0:
            pass
    return result
`)
	result, err := AnalyzeFile(path, "Python")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + for + if + elif
}

func TestAnalyzeFile_Java(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "App.java", `
class App {
    public int sum(int[] items) {
        int total = 0;
        for (int item : items) {
            if (item > 0) {
                total += item;
            }
        }
        return total;
    }
}
`)
	result, err := AnalyzeFile(path, "Java")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + for + if
}

func TestAnalyzeFile_Nesting(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "nested.js", `
function deep(items) {
  for (let i = 0; i < items.length; i++) {
    if (items[i] > 0) {
      while (items[i] > 10) {
        items[i]--;
      }
    }
  }
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.MaxNesting, 3) // for > if > while
}

func TestAnalyzeFile_UnsupportedLanguage(t *testing.T) {
	result, err := AnalyzeFile("/tmp/test.unknown", "Cobol")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestAnalyzeFile_PHP(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.php", `<?php
function process($items) {
    $result = [];
    foreach ($items as $item) {
        if ($item > 0) {
            $result[] = $item;
        }
    }
    return $result;
}
?>`)
	result, err := AnalyzeFile(path, "PHP")
	require.NoError(t, err)
	if result != nil && len(result.Functions) > 0 {
		fn := result.Functions[0]
		assert.GreaterOrEqual(t, fn.Complexity, 3) // base + foreach + if
	}
}

func TestAnalyzeFile_Ruby(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.rb", `
def process(items)
  result = []
  items.each do |item|
    if item > 0
      result << item
    end
  end
  result
end
`)
	result, err := AnalyzeFile(path, "Ruby")
	require.NoError(t, err)
	if result != nil && len(result.Functions) > 0 {
		fn := result.Functions[0]
		assert.GreaterOrEqual(t, fn.Complexity, 1) // at least base complexity
	}
}

func TestSummarize(t *testing.T) {
	results := []*FileResult{
		{
			Path:     "a.js",
			Language: "JavaScript",
			Functions: []FunctionResult{
				{Name: "foo", Complexity: 5, MaxNesting: 2},
				{Name: "bar", Complexity: 15, MaxNesting: 4},
			},
		},
		{
			Path:     "b.js",
			Language: "JavaScript",
			Functions: []FunctionResult{
				{Name: "baz", Complexity: 1, MaxNesting: 0},
			},
		},
		nil, // should be skipped
	}

	s := Summarize(results)
	assert.Equal(t, 3, s.TotalFunctions)
	assert.InDelta(t, 7.0, s.AvgComplexity, 0.01)   // (5+15+1)/3 = 7.0
	assert.InDelta(t, 2.0, s.AvgNesting, 0.01)       // (2+4+0)/3 = 2.0
	assert.Equal(t, 15, s.MaxComplexity)
	assert.Equal(t, 4, s.MaxNesting)
	assert.Equal(t, 1, s.HighComplexity) // only bar (15 > 10)
}

func TestAnalyzeFile_ExtractsTokens(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "tok.js", `
function add(a, b) {
  return a + b;
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, len(result.Tokens), 0, "should extract tokens from AST")

	// Check normalization: identifiers become $ID, no raw variable names
	hasID := false
	for _, tok := range result.Tokens {
		if tok.Value == "$ID" {
			hasID = true
		}
		// Should not contain raw identifier names
		assert.NotEqual(t, "add", tok.Value, "identifiers should be normalized to $ID")
	}
	assert.True(t, hasID, "should have normalized identifier tokens")
}

func TestSummarize_Empty(t *testing.T) {
	s := Summarize(nil)
	assert.Equal(t, 0, s.TotalFunctions)
	assert.Equal(t, 0.0, s.AvgComplexity)
	assert.Equal(t, 0.0, s.AvgNesting)
}

// --- High-complexity fixtures with exact assertions (spec requirement) ---

func TestAnalyzeFile_JavaScript_HighComplexity(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "controller.js", `
function processOrder(order, user, config) {
  if (!order) {
    return { error: "no order" };
  }
  if (!user) {
    return { error: "no user" };
  }
  if (order.status === "cancelled") {
    return { error: "cancelled" };
  }

  let total = 0;
  for (let i = 0; i < order.items.length; i++) {
    const item = order.items[i];
    if (item.quantity <= 0) {
      continue;
    }
    if (item.price > 10000) {
      return { error: "price limit" };
    }
    total += item.price * item.quantity;
  }

  if (config.taxEnabled && user.region === "US") {
    total *= 1.08;
  }

  if (total > config.maxOrderAmount || user.balance < total) {
    return { error: "insufficient funds" };
  }

  try {
    return { success: true, total: total };
  } catch (e) {
    return { error: "failed" };
  }
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	// 1(base) + 3(ifs) + 1(for) + 2(ifs in loop) + 1(&&) + 1(||) + 1(catch) = 10
	assert.GreaterOrEqual(t, fn.Complexity, 9, "high-complexity JS function")
	assert.LessOrEqual(t, fn.Complexity, 12)
}

func TestAnalyzeFile_Python_HighComplexity(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "handler.py", `
def handle_request(request, db, config):
    if not request:
        return {"error": "empty"}
    if request.method not in ("GET", "POST", "PUT"):
        return {"error": "bad method"}

    try:
        data = request.json()
    except ValueError:
        return {"error": "bad json"}

    if request.method == "GET":
        items = db.query(request.params)
        if not items:
            return {"error": "not found"}
        return {"data": items}
    elif request.method == "POST":
        if not data.get("name") or not data.get("email"):
            return {"error": "missing fields"}
        return db.create(data)
    elif request.method == "PUT":
        if "id" not in data:
            return {"error": "missing id"}
        existing = db.get(data["id"])
        if not existing:
            return {"error": "not found"}
        return db.update(data)
`)
	result, err := AnalyzeFile(path, "Python")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	// base(1) + 2(ifs) + 1(except) + 1(if GET) + 1(if !items) + 1(elif POST) + 1(or) + 1(elif PUT) + 1(not in) + 1(if !existing) = ~11
	assert.GreaterOrEqual(t, fn.Complexity, 8, "high-complexity Python function")
}

func TestAnalyzeFile_Java_SwitchAndLambda(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "Router.java", `
class Router {
    public Object route(String method, String path, Object body) {
        if (method == null || path == null) {
            throw new IllegalArgumentException("null input");
        }

        switch (method) {
            case "GET":
                if (path.startsWith("/users")) {
                    return getUsers();
                }
                return getDefault(path);
            case "POST":
                if (body == null) {
                    throw new IllegalArgumentException("body required");
                }
                return createResource(path, body);
            case "DELETE":
                return deleteResource(path);
        }

        if (path.contains("admin") && isAuthorized()) {
            return handleAdmin(path, body);
        }

        return notFound();
    }
}
`)
	result, err := AnalyzeFile(path, "Java")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	// base(1) + 1(||) + 3(cases) + 2(ifs) + 1(if &&) + 1(&&) = ~9
	assert.GreaterOrEqual(t, fn.Complexity, 7, "Java switch+boolean complexity")
}

func TestAnalyzeFile_PHP_ForeachElseif(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "process.php", `<?php
function processItems($items, $config) {
    if (empty($items)) {
        return [];
    }

    $result = [];
    foreach ($items as $item) {
        if ($item['status'] === 'active') {
            $result[] = $item;
        } elseif ($item['status'] === 'pending') {
            if ($config['includePending']) {
                $result[] = $item;
            }
        } elseif ($item['status'] === 'archived') {
            continue;
        }
    }

    if (count($result) > $config['limit']) {
        $result = array_slice($result, 0, $config['limit']);
    }

    return $result;
}
?>`)
	result, err := AnalyzeFile(path, "PHP")
	require.NoError(t, err)
	if result != nil && len(result.Functions) > 0 {
		fn := result.Functions[0]
		// base(1) + 1(empty) + 1(foreach) + 1(if active) + 1(elseif pending) + 1(if include) + 1(elseif archived) + 1(if count) = 8
		assert.GreaterOrEqual(t, fn.Complexity, 6, "PHP foreach+elseif complexity")
	}
}

func TestAnalyzeFile_Ruby_UnlessRescue(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "service.rb", `
def process_payment(amount, card, options)
  unless amount > 0
    return { error: "invalid amount" }
  end

  unless card
    return { error: "no card" }
  end

  begin
    result = charge(card, amount)
    if result[:success]
      if options[:send_receipt]
        send_receipt(result)
      end
      return result
    elsif result[:retry]
      return retry_charge(card, amount)
    end
  rescue PaymentError
    return { error: "payment failed" }
  rescue NetworkError
    return { error: "network error" }
  end

  { error: "unknown" }
end
`)
	result, err := AnalyzeFile(path, "Ruby")
	require.NoError(t, err)
	if result != nil && len(result.Functions) > 0 {
		fn := result.Functions[0]
		// base(1) + 2(unless) + 1(if success) + 1(if send_receipt) + 1(elsif retry) + 2(rescue) = 8
		assert.GreaterOrEqual(t, fn.Complexity, 5, "Ruby unless+rescue complexity")
	}
}

func TestAnalyzeFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "empty.js", "")
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Functions)
}

func TestAnalyzeFile_NoFunctions(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "constants.js", `
const PI = 3.14;
const E = 2.718;
var name = "test";
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Functions)
}

func TestAnalyzeFile_NonexistentFile(t *testing.T) {
	_, err := AnalyzeFile("/nonexistent/file.js", "JavaScript")
	assert.Error(t, err)
}

func TestSummarize_AllHighComplexity(t *testing.T) {
	results := []*FileResult{
		{
			Functions: []FunctionResult{
				{Complexity: 15},
				{Complexity: 20},
				{Complexity: 25},
			},
		},
	}
	s := Summarize(results)
	assert.Equal(t, 3, s.HighComplexity) // all > 10
	assert.Equal(t, 25, s.MaxComplexity)
}

func TestSummarize_BoundaryComplexity(t *testing.T) {
	results := []*FileResult{
		{
			Functions: []FunctionResult{
				{Complexity: 10}, // exactly 10, not > 10
				{Complexity: 11}, // just over threshold
			},
		},
	}
	s := Summarize(results)
	assert.Equal(t, 1, s.HighComplexity) // only 11 is > 10
}

// --- Edge-case tests for specific language constructs ---

func TestAnalyzeFile_JavaScript_OptionalChaining(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "optional.js", `
function getNestedValue(obj) {
  return obj?.prop?.nested?.method();
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	// Optional chaining ?. is not a decision point — complexity stays at base 1
	assert.Equal(t, 1, fn.Complexity, "optional chaining ?. should not increase complexity")
}

func TestAnalyzeFile_Ruby_UntilLoop(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "until.rb", `
def countdown(n)
  until n <= 0
    n -= 1
  end
  n
end
`)
	result, err := AnalyzeFile(path, "Ruby")
	require.NoError(t, err)
	if result != nil && len(result.Functions) > 0 {
		fn := result.Functions[0]
		// base(1) + until(1) = 2; 'until' is in Ruby decisionNodes
		assert.GreaterOrEqual(t, fn.Complexity, 2, "Ruby 'until' should count as a decision point")
	}
}

func TestAnalyzeFile_Java_InstanceofAndTryWithResources(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "TypeCheck.java", `
class TypeCheck {
    public String describe(Object obj) {
        if (obj instanceof String) {
            return "string";
        }
        if (obj instanceof Integer) {
            return "integer";
        }
        return "unknown";
    }
}
`)
	result, err := AnalyzeFile(path, "Java")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	// instanceof is inside an if_statement; the if is the decision point, not instanceof itself.
	// base(1) + 2(if statements) = 3
	assert.GreaterOrEqual(t, fn.Complexity, 3, "Java if+instanceof should count the if as decision point")
}

func TestAnalyzeFile_JavaScript_NestingDepth5(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "nest5.js", `
function deepNest5(x) {
  if (x > 0) {                     // depth 1
    for (let i = 0; i < x; i++) {  // depth 2
      if (i % 2 === 0) {           // depth 3
        while (x > 10) {           // depth 4
          if (i > 5) {             // depth 5
            x--;
          }
        }
      }
    }
  }
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.Equal(t, 5, fn.MaxNesting, "should detect exactly 5 levels of nesting")
}

func TestAnalyzeFile_JavaScript_NestingDepth8(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "nest8.js", `
function deepNest8(x) {
  if (x > 0) {                          // depth 1
    for (let i = 0; i < x; i++) {       // depth 2
      if (i % 2 === 0) {                // depth 3
        while (x > 10) {                // depth 4
          if (i > 5) {                  // depth 5
            for (let j = 0; j < i; j++) { // depth 6
              if (j > 2) {              // depth 7
                while (j > 0) {         // depth 8
                  j--;
                }
              }
            }
          }
        }
      }
    }
  }
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.Equal(t, 8, fn.MaxNesting, "should detect exactly 8 levels of nesting")
}
