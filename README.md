# WASA — Full-Stack Messaging Application

A full-stack web messaging application built with **Go** and **Vue.js**, featuring a RESTful backend API and a responsive web client.

The application implements the core functionality of a modern messaging platform, including user management, conversations, and real-time-style message interactions. The project focuses on clean API design, client-server communication, and modular web application architecture.

## Features

- User registration and profile management
- Create and manage conversations
- Send and receive messages
- RESTful client-server communication
- Interactive web-based chat interface
- Modular backend architecture
- Responsive frontend interface

## Tech Stack

### Backend

- **Go (Golang)**
- **RESTful API**
- **Go Modules**

### Frontend

- **Vue.js**
- **JavaScript**
- **Bootstrap**

### Development & Tooling

- **Docker**
- **Node.js**
- **Yarn**
- **Git**

## Architecture

The application follows a client-server architecture with a clear separation between the backend API and frontend interface.

```text
┌─────────────────────┐
│     Vue.js Client   │
│   Web Interface     │
└──────────┬──────────┘
           │
           │ REST API
           ▼
┌─────────────────────┐
│      Go Backend     │
│   API & Services    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│    Application Data │
└─────────────────────┘
```

The backend is responsible for application logic and API endpoints, while the Vue.js frontend provides the user interface and communicates with the backend through HTTP requests.

## Project Structure

```text
Wasa/
├── cmd/          # Application entry points
├── service/      # Backend services and API logic
├── webui/        # Vue.js frontend
├── doc/          # API documentation
├── demo/         # Demo resources
├── go.mod
├── Dockerfile.backend
└── Dockerfile.frontend
```

## About

This project was originally developed as part of the **Web and Software Architecture (WASA)** course at **Sapienza University of Rome** and was further developed as a full-stack software engineering project.

Its main focus is the practical implementation of REST API design, backend/frontend separation, and modern web application architecture.
