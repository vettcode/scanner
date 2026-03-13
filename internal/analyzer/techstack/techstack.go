package techstack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Result holds the tech stack detection results.
type Result struct {
	Frameworks []string
	Runtimes   []RuntimeInfo
	Databases  []string
	Services   []string
}

// RuntimeInfo holds a detected runtime and its version.
type RuntimeInfo struct {
	Name    string
	Version string // may be empty if not detected
}

// frameworkPatterns maps dependency names to framework names.
var frameworkPatterns = map[string]string{
	// JavaScript/TypeScript
	"react":                "React",
	"react-dom":            "React",
	"next":                 "Next.js",
	"nuxt":                 "Nuxt.js",
	"vue":                  "Vue.js",
	"@angular/core":        "Angular",
	"svelte":               "Svelte",
	"express":              "Express",
	"fastify":              "Fastify",
	"koa":                  "Koa",
	"hapi":                 "Hapi",
	"nestjs":               "NestJS",
	"@nestjs/core":         "NestJS",
	"gatsby":               "Gatsby",
	"remix":                "Remix",
	"electron":             "Electron",
	// Python
	"django":               "Django",
	"flask":                "Flask",
	"fastapi":              "FastAPI",
	"tornado":              "Tornado",
	"celery":               "Celery",
	"aiohttp":              "aiohttp",
	"starlette":            "Starlette",
	"streamlit":            "Streamlit",
	// PHP
	"laravel/framework":    "Laravel",
	"symfony/symfony":       "Symfony",
	"slim/slim":            "Slim",
	"cakephp/cakephp":      "CakePHP",
	// Ruby
	"rails":                "Ruby on Rails",
	"sinatra":              "Sinatra",
	"hanami":               "Hanami",
	// Java
	"org.springframework":  "Spring",
	"spring-boot":          "Spring Boot",
	// Go
	"github.com/gin-gonic/gin":    "Gin",
	"github.com/gorilla/mux":      "Gorilla Mux",
	"github.com/labstack/echo":    "Echo",
	"github.com/gofiber/fiber":    "Fiber",
}

// databasePatterns maps dependency names to database names.
var databasePatterns = map[string]string{
	// SQL
	"pg":                     "PostgreSQL",
	"postgres":               "PostgreSQL",
	"psycopg2":               "PostgreSQL",
	"asyncpg":                "PostgreSQL",
	"mysql":                  "MySQL",
	"mysql2":                 "MySQL",
	"pymysql":                "MySQL",
	"sqlite3":                "SQLite",
	"better-sqlite3":         "SQLite",
	"sequelize":              "Sequelize (ORM)",
	"prisma":                 "Prisma (ORM)",
	"@prisma/client":         "Prisma (ORM)",
	"typeorm":                "TypeORM (ORM)",
	"drizzle-orm":            "Drizzle (ORM)",
	"sqlalchemy":             "SQLAlchemy (ORM)",
	"peewee":                 "Peewee (ORM)",
	"django":                 "Django ORM",
	"activerecord":           "Active Record (ORM)",
	// NoSQL
	"mongodb":                "MongoDB",
	"mongoose":               "MongoDB",
	"pymongo":                "MongoDB",
	"redis":                  "Redis",
	"ioredis":                "Redis",
	"elasticsearch":          "Elasticsearch",
	"@elastic/elasticsearch": "Elasticsearch",
	"cassandra-driver":       "Cassandra",
	"dynamodb":               "DynamoDB",
	"@aws-sdk/client-dynamodb": "DynamoDB",
	"firebase":               "Firebase",
	"firebase-admin":         "Firebase",
	"supabase":               "Supabase",
	"@supabase/supabase-js":  "Supabase",
}

// servicePatterns maps dependency names to external service names.
var servicePatterns = map[string]string{
	"stripe":                   "Stripe",
	"@stripe/stripe-js":        "Stripe",
	"@sendgrid/mail":           "SendGrid",
	"sendgrid":                 "SendGrid",
	"twilio":                   "Twilio",
	"aws-sdk":                  "AWS SDK",
	"@aws-sdk/client-s3":       "AWS S3",
	"@aws-sdk/client-ses":      "AWS SES",
	"boto3":                    "AWS SDK (boto3)",
	"google-cloud-storage":     "Google Cloud Storage",
	"@google-cloud/storage":    "Google Cloud Storage",
	"@azure/storage-blob":      "Azure Blob Storage",
	"resend":                   "Resend",
	"postmark":                 "Postmark",
	"mailgun":                  "Mailgun",
	"pusher":                   "Pusher",
	"@clerk/clerk-sdk-node":    "Clerk",
	"@clerk/nextjs":            "Clerk",
	"@auth0/auth0-spa-js":      "Auth0",
	"auth0":                    "Auth0",
	"passport":                 "Passport.js",
	"cloudinary":               "Cloudinary",
	"@upstash/redis":           "Upstash",
	"plaid":                    "Plaid",
	"braintree":                "Braintree",
}

// Detect detects the tech stack from dependency names and the project root.
func Detect(root string, deps []string) *Result {
	r := &Result{}
	frameworkSet := make(map[string]bool)
	dbSet := make(map[string]bool)
	svcSet := make(map[string]bool)

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		if fw, ok := frameworkPatterns[dep]; ok {
			frameworkSet[fw] = true
		} else if fw, ok := frameworkPatterns[depLower]; ok {
			frameworkSet[fw] = true
		}

		if db, ok := databasePatterns[dep]; ok {
			dbSet[db] = true
		} else if db, ok := databasePatterns[depLower]; ok {
			dbSet[db] = true
		}

		if svc, ok := servicePatterns[dep]; ok {
			svcSet[svc] = true
		} else if svc, ok := servicePatterns[depLower]; ok {
			svcSet[svc] = true
		}
	}

	r.Frameworks = setToSortedSlice(frameworkSet)
	r.Databases = setToSortedSlice(dbSet)
	r.Services = setToSortedSlice(svcSet)
	r.Runtimes = detectRuntimes(root)

	return r
}

// detectRuntimes checks for runtime version files.
func detectRuntimes(root string) []RuntimeInfo {
	var runtimes []RuntimeInfo

	// .nvmrc or .node-version → Node.js version
	for _, f := range []string{".nvmrc", ".node-version"} {
		if v := readTrimmedFile(filepath.Join(root, f)); v != "" {
			runtimes = append(runtimes, RuntimeInfo{Name: "Node.js", Version: strings.TrimPrefix(v, "v")})
			break
		}
	}

	// .python-version → Python version
	if v := readTrimmedFile(filepath.Join(root, ".python-version")); v != "" {
		runtimes = append(runtimes, RuntimeInfo{Name: "Python", Version: v})
	}

	// .ruby-version → Ruby version
	if v := readTrimmedFile(filepath.Join(root, ".ruby-version")); v != "" {
		runtimes = append(runtimes, RuntimeInfo{Name: "Ruby", Version: v})
	}

	// .go-version or go.mod → Go version
	if v := readTrimmedFile(filepath.Join(root, ".go-version")); v != "" {
		runtimes = append(runtimes, RuntimeInfo{Name: "Go", Version: strings.TrimPrefix(v, "go")})
	} else if v := goModVersion(filepath.Join(root, "go.mod")); v != "" {
		runtimes = append(runtimes, RuntimeInfo{Name: "Go", Version: v})
	}

	// .java-version or pom.xml → Java version
	if v := readTrimmedFile(filepath.Join(root, ".java-version")); v != "" {
		runtimes = append(runtimes, RuntimeInfo{Name: "Java", Version: v})
	}

	// package.json engines.node → Node.js (fallback)
	if len(runtimes) == 0 || !hasRuntime(runtimes, "Node.js") {
		if v := nodeEngineVersion(filepath.Join(root, "package.json")); v != "" {
			runtimes = append(runtimes, RuntimeInfo{Name: "Node.js", Version: v})
		}
	}

	return runtimes
}

func readTrimmedFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func goModVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimPrefix(line, "go ")
		}
	}
	return ""
}

func nodeEngineVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if json.Unmarshal(data, &pkg) == nil && pkg.Engines.Node != "" {
		return pkg.Engines.Node
	}
	return ""
}

func hasRuntime(runtimes []RuntimeInfo, name string) bool {
	for _, r := range runtimes {
		if r.Name == name {
			return true
		}
	}
	return false
}

func setToSortedSlice(s map[string]bool) []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
