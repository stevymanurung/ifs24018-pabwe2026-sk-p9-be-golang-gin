pipeline {
  agent any
  stages {
    stage('Checkout') { steps { checkout scm } }
    stage('Backend Tests') {
      steps { sh 'go test -v -cover ./internal/...' }
    }
    stage('Backend Build') {
      steps { sh 'go build -o app ./cmd/server' }
    }
  }
  post { always { archiveArtifacts artifacts: 'app', allowEmptyArchive: true } }
}
