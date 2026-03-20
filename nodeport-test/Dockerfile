FROM python:3.9-alpine
WORKDIR /app
RUN echo "Hello from simple-webapp!" > index.html
EXPOSE 8080
CMD ["python", "-m", "http.server", "8080"]
