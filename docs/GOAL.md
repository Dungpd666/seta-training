# 🔥 Microservices Challenge: User, Team & Asset Management

## 🎯 Project Intention & Introduction
Welcome to the Capstone Mini-Project for the 2026 Golang Intern Course! 

The goal of this project is to evaluate your system design, coding, and problem-solving skills in building a robust backend system. Unlike standard tutorials, **this project is technology-agnostic.** While the core language is Go (to align with the course), you have the absolute freedom to choose your preferred API protocols (REST, GraphQL, gRPC), databases (SQL vs. NoSQL), frameworks, and infrastructure tools. 

You are expected to research your choices, justify your trade-offs, and implement best practices for a microservices architecture. The project will be released in **3 progressive stages**, allowing you to continuously build, iterate, and scale the system alongside the course syllabus.

> 💡 *"Every technical decision should be a deliberate trade-off, balancing the pragmatism of today with the scale of tomorrow."*

## 👩🏻‍💻 System Overview
You are tasked with building a microservices-based system to manage users, teams, and digital assets. 
* **Users** can have varying roles (Manager or Member).
* **Managers** can form teams and manage personnel.
* **All Users** can manage, organize, and share digital assets (folders & notes) with granular access control.

## 📋 General Requirements

### 1. General Engineering Standards
* **Authentication & Authorization:** Secure all endpoints and validate user roles before executing sensitive actions.
* **Clean Architecture:** Structure your code utilizing separation of concerns (e.g., handlers, services, repositories).
* **Error Handling:** Ensure graceful degradation and use appropriate HTTP/RPC status codes.

### 2. Technical Documentation
* You must prepare a **500–800 word document** describing the tech stack used in your project.
* Detail why you chose your specific database, message queue, or API protocol. What were the trade-offs, and how do they benefit your specific implementation?

### 3. Final Presentation (Demo & Future Pitch)
* You will participate in a live **3-minute presentation**.
* Pitch **one major future improvement** for your system. If this were a real startup, what is the next technical bottleneck you would hit, and how would you re-architect the system to solve it?

*(Note: The detailed technical specifications for the project are divided into Stage 1, Stage 2, and Stage 3 documents. All submissions must be in English).*
