pipeline {
    environment {
        specName = 'pmm-doc'
    }
    agent {
        label 'micro-amazon'
    }
    stages {
        stage('Prepare') {
            steps {
                sh '''
                    sudo yum -y install git wget docker
                    sudo usermod -aG docker `id -u -n`
                    sudo service docker start
                '''
            }
        }
        stage('Build Docs') {
            steps {
                sh '''
                    sudo make -C doc clean
                    make -C doc theme
                    sg docker -c "
                        docker run --rm -v $(pwd -P)/doc:/doc florianholzapfel/docker-alpine-sphinx make -C /doc html
                    "
                '''
            }
        }
        stage('Publish Docs') {
            steps {
                withCredentials([[$class: 'AmazonWebServicesCredentialsBinding', accessKeyVariable: 'AWS_ACCESS_KEY_ID', credentialsId: 'AMI/OVF', secretKeyVariable: 'AWS_SECRET_ACCESS_KEY']]) {
                    sh '''
                        aws s3 sync \
                            --region eu-west-1 \
                            --delete \
                            ./doc/build/html/ \
                            s3://docs-test.cd.percona.com/${GIT_BRANCH}/
                    '''
                }
            }
        }
    }

    post {
        always {
            script {
                if (currentBuild.result == null || currentBuild.result == 'SUCCESS') {
                    if (env.CHANGE_URL) {
                        withCredentials([string(credentialsId: 'GITHUB_API_TOKEN', variable: 'GITHUB_API_TOKEN')]) {
                            sh """
                                set -o xtrace
                                curl -v -X POST \
                                    -H "Authorization: token ${GITHUB_API_TOKEN}" \
                                    -d "{\\"body\\":\\"http://docs-test.cd.percona.com/${GIT_BRANCH}/\\"}" \
                                    "https://api.github.com/repos/\$(echo $CHANGE_URL | cut -d '/' -f 4-5)/issues/${CHANGE_ID}/comments"
                            """
                        }
                    }
                    slackSend channel: '#pmm-ci', color: '#00FF00', message: "[${specName}]: build finished - http://docs-test.cd.percona.com/${GIT_BRANCH}/"
                } else {
                    slackSend channel: '#pmm-ci', color: '#FF0000', message: "[${specName}]: build ${currentBuild.result}"
                }
            }
            sh 'sudo rm -rf *'
            deleteDir()
        }
    }
}
