package compose

import "fmt"

type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var composeTemplates = map[string]string{
	"nginx": `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    restart: unless-stopped
`,
	"mysql": `services:
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: changeme
      MYSQL_DATABASE: app
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    restart: unless-stopped
volumes:
  mysql_data:
`,
	"redis": `services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    restart: unless-stopped
`,
	"wordpress": `services:
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: changeme
      MYSQL_DATABASE: wordpress
    volumes:
      - db_data:/var/lib/mysql
    restart: unless-stopped
  wordpress:
    image: wordpress:latest
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: root
      WORDPRESS_DB_PASSWORD: changeme
      WORDPRESS_DB_NAME: wordpress
    depends_on:
      - db
    restart: unless-stopped
volumes:
  db_data:
`,
	"portainer": `services:
  portainer:
    image: portainer/portainer-ce:latest
    ports:
      - "9000:9000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer_data:/data
    restart: unless-stopped
volumes:
  portainer_data:
`,
}

func ListTemplates() []Template {
	return []Template{
		{ID: "nginx", Name: "Nginx", Description: "Static web server on port 8080"},
		{ID: "mysql", Name: "MySQL 8", Description: "MySQL database with persistent volume"},
		{ID: "redis", Name: "Redis 7", Description: "In-memory cache on port 6379"},
		{ID: "wordpress", Name: "WordPress", Description: "WordPress + MySQL stack on port 8080"},
		{ID: "portainer", Name: "Portainer CE", Description: "Docker management UI on port 9000"},
	}
}

func TemplateYAML(id string) (string, error) {
	if id == "" || id == "nginx" {
		return composeTemplates["nginx"], nil
	}
	yaml, ok := composeTemplates[id]
	if !ok {
		return "", fmt.Errorf("未知模板: %s", id)
	}
	return yaml, nil
}
