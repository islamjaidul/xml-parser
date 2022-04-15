FROM golang:1.16

# Install the air binary so we get live code-reloading when we save files
# RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# WORKDIR /opt/app/api

# CMD ["air"]