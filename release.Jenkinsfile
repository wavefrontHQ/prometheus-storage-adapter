pipeline {

    agent any

    tools {
        go 'Go 1.18'
    }

    environment {
        RELEASE_TYPE = 'release'
        GIT_CREDENTIAL_ID = 'wf-jenkins-github'
        GITHUB_TOKEN = credentials('GITHUB_TOKEN')
        REPO_NAME = 'prometheus-storage-adapter'
    }

    parameters {
        string(name: 'VERSION_NUMBER', defaultValue: '', description: 'The version number to release as')
        string(name: 'TARGET_COMITISH', defaultValue: 'master', description: 'Specify a specific commit hash or branch to release the version to.')
        string(name: 'RELEASE_NOTES', defaultValue: '', description: 'The public release notes for the version. Use \\n to create newlines')
        booleanParam(name: 'IS_DRAFT', defaultValue: true, description: 'If the release should be marked as a draft (unpublished)')
        booleanParam(name: 'IS_PRERELEASE', defaultValue: false, description: 'If the release should be marked as a prerelease')
        booleanParam(name: 'createGithubRelease', defaultValue: true, description: 'Mark as false if you only want to build/push docker images. Note: a tag specified by the name VERSION_NUMBER must be created in the repo already for this option to be turned off')
    }

    stages {
        stage('Build Linux Binary') {
            steps {
                script {
                    sh 'make build-linux'
                }
            }
        }

        stage('Create Github Release') {
            when {
                beforeAgent true
                expression { return params.createGithubRelease }
            }
            environment {
                VERSION_NUMBER = "${params.VERSION_NUMBER}"
                TARGET_COMITISH_TRIMMED = TARGET_COMITISH.minus("origin/")
                RELEASE_NOTES = "${params.RELEASE_NOTES}"
                IS_DRAFT = "${params.IS_DRAFT}"
                IS_PRERELEASE = "${params.IS_PRERELEASE}"
            }
            steps {
                sh 'curl -X POST -H \"Authorization: token ${GITHUB_TOKEN}\" -H \"Accept: application/vnd.github.v3+json\" https://api.github.com/repos/wavefrontHQ/${REPO_NAME}/releases -d \"{\\"tag_name\\": \\"${VERSION_NUMBER}\\", \\"target_commitish\\": \\"${TARGET_COMITISH_TRIMMED}\\", \\"name\\": \\"Release ${VERSION_NUMBER}\\", \\"body\\": \\"${RELEASE_NOTES}\\", \\"draft\\": ${IS_DRAFT}, \\"prerelease\\": ${IS_PRERELEASE}}\" '
            }
        }
        /*
        stage('Make/Push Docker Image') {
            steps {
                script {
                    docker.withRegistry('https://projects.registry.vmware.com', 'projects-registry-vmware-tanzu_observability-robot') {
                        def harborImage = docker.build("tanzu_observability/${REPO_NAME}:${VERSION_NUMBER}")
                        harborImage.push()
                        harborImage.push("latest")
                    }

                    docker.withRegistry('https://index.docker.io/v1/', 'Dockerhub_svcwfjenkins') {
                      def dockerHubImage = docker.build("wavefronthq/${REPO_NAME}:${VERSION_NUMBER}")
                      dockerHubImage.push()
                      dockerHubImage.push("latest")
                    }
                }
            }
        }
        */
    }

    post {
      // Notify only on null->failure or success->failure or any->success
      /*
      failure {
        script {
          if(currentBuild.previousBuild == null) {
            slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
          }
        }
      }
      regression {
        slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
      }
      success {
        script {
          slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "Success!! `prometheus-storage-adapter:${VERSION_NUMBER}` released!")
        }
      }
      */
      always {
        cleanWs()
      }
    }
}
