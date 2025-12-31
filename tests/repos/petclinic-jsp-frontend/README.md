# PetClinic JSP Frontend

A Java/JSP frontend application for the Spring PetClinic REST API. This project demonstrates a traditional server-side rendered web application using Jakarta Servlets and JSP technology.

## Features

- **Owner Management**: Create, view, edit, and delete pet owners
- **Pet Management**: Add and manage pets for each owner
- **Visit Tracking**: Record and view veterinary visits for pets
- **Veterinarian Directory**: Browse veterinarians and their specialties
- **Search Functionality**: Search owners by last name
- **Responsive Design**: Bootstrap 5 based UI

## Architecture

```
petclinic-jsp-frontend/
├── src/main/java/com/example/petclinic/
│   ├── model/          # DTOs (OwnerDto, PetDto, VetDto, VisitDto, PetTypeDto)
│   ├── service/        # API Client for REST communication
│   └── servlet/        # Servlet controllers
├── src/main/webapp/
│   ├── WEB-INF/
│   │   ├── jsp/        # JSP view templates
│   │   │   ├── includes/   # Header, footer, nav
│   │   │   ├── owners/     # Owner views
│   │   │   ├── pets/       # Pet views
│   │   │   ├── visits/     # Visit views
│   │   │   └── vets/       # Vet views
│   │   └── web.xml     # Web application configuration
│   ├── css/            # Stylesheets
│   └── js/             # JavaScript files
└── pom.xml             # Maven configuration
```

## Prerequisites

- Java 17 or higher
- Maven 3.6 or higher
- Running Spring PetClinic REST API (default: http://localhost:9966/petclinic/api)

## Configuration

The API base URL can be configured in `src/main/webapp/WEB-INF/web.xml`:

```xml
<context-param>
    <param-name>apiBaseUrl</param-name>
    <param-value>http://localhost:9966/petclinic/api</param-value>
</context-param>
```

## Building

```bash
mvn clean package
```

This creates a WAR file at `target/petclinic-frontend.war`.

## Running

### Using Embedded Tomcat (Development)

```bash
mvn cargo:run
```

Access the application at http://localhost:8080/petclinic-frontend/

### Deploying to Tomcat

Copy `target/petclinic-frontend.war` to your Tomcat's `webapps` directory.

## Technology Stack

- **Jakarta Servlet 6.0**: Servlet API for request handling
- **Jakarta JSP 3.1**: JavaServer Pages for view rendering
- **JSTL 3.0**: JSP Standard Tag Library
- **Gson**: JSON processing for REST API communication
- **Java HTTP Client**: Built-in HTTP client for API calls
- **Bootstrap 5**: CSS framework for responsive design
- **Bootstrap Icons**: Icon library

## REST API Endpoints Used

| Endpoint | Description |
|----------|-------------|
| GET /owners | List all owners |
| GET /owners/{id} | Get owner by ID |
| POST /owners | Create new owner |
| PUT /owners/{id} | Update owner |
| DELETE /owners/{id} | Delete owner |
| GET /owners/{id}/pets | List owner's pets |
| POST /owners/{id}/pets | Add pet to owner |
| GET /vets | List all veterinarians |
| GET /pettypes | List pet types |
| POST /owners/{id}/pets/{petId}/visits | Create visit |

## Flow Diagram

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Browser    │────▶│   Servlet    │────▶│  API Client  │
│              │◀────│  (Controller)│◀────│              │
└──────────────┘     └──────────────┘     └──────────────┘
                            │                     │
                            ▼                     ▼
                     ┌──────────────┐     ┌──────────────┐
                     │     JSP      │     │  PetClinic   │
                     │   (View)     │     │  REST API    │
                     └──────────────┘     └──────────────┘
```

## License

This project is for educational purposes.
