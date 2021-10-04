pipeline {
    agent any

    tools {
        go 'Go 1.15'
    }

    environment {
        VERSION = "${params.VERSION}"
    }

    stages {
      stage("Build & Publish") {
        environment {
            DOCKERHUB_CREDS=credentials('Dockerhub_svcwfjenkins')
        }
        steps {
          sh 'echo $DOCKERHUB_CREDS_PSW | docker login -u $DOCKERHUB_CREDS_USR --password-stdin'
          sh 'make publish'
        }
      }
    }
}
