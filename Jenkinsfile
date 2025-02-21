pipeline {
    agent any

    stages {
        stage('Test') {
            steps {
                echo "Testing the app..."
            }
        }
        stage('Prepare') {
            steps {
                script {
                    echo "Preparing app, running on branch ${env.BRANCH_NAME}"
                }
            }
        }
        stage('Build') {
            steps {
                script {
                    // Build docker images
                    sh 'docker compose -f docker-compose.yml build'
                }
            }
        }
        stage('Deploy') {
            steps {
                withCredentials([file(credentialsId: 'diabetify-be-env', variable: 'ENV_FILE')]) {
                    script {
                        sh '''
                            cat "${ENV_FILE}" > .env

                            echo "Using the following .env contents:"
                            cat .env

                            docker compose -f docker-compose.yml up -d
                        '''
                    }
                }
                echo "Deploying the app completed."
            }
        }
    }
}