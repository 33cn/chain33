pipeline {
    agent any

    environment {
        GOPATH = "${WORKSPACE}"
        PROJ_DIR = "${WORKSPACE}/src/gitlab.33.cn/chain33/chain33"
    }

    options {
        timeout(time: 2,unit: 'HOURS')
        retry(1)
        timestamps()
        gitLabConnection('gitlab33')
        gitlabBuilds(builds: ['check', 'build', 'test', 'deploy'])
        checkoutToSubdirectory "src/gitlab.33.cn/chain33/chain33"
    }

    stages {
        stage('check') {
            steps {
                dir("${PROJ_DIR}"){
                    gitlabCommitStatus(name: 'check'){
                        sh "make auto_ci branch=${env.gitlabSourceBranch}"
                    }
                }
            }
        }

        stage('build') {
            steps {
                dir("${env.PROJ_DIR}"){
                    gitlabCommitStatus(name: 'build'){
                        sh 'make checkgofmt'
                        sh 'make linter'
                    }
                }
            }
        }

        stage('test'){
            agent {
                docker{
                    image 'suyanlong/chain33-run:latest'
                }
            }

            environment {
                GOPATH = "${WORKSPACE}"
                PROJ_DIR = "${WORKSPACE}/src/gitlab.33.cn/chain33/chain33"
            }

            steps {
                dir("${env.PROJ_DIR}"){
                    gitlabCommitStatus(name: 'test'){
                        sh 'make test'
                        //sh 'export CC=clang-5.0 && make msan'
                    }
                }
            }
        }

        stage('deploy') {
            steps {
                dir("${PROJ_DIR}"){
                    gitlabCommitStatus(name: 'deploy'){
                        sh 'make build_ci'
                        sh "make autotest_ci proj=${env.BUILD_NUMBER}"
                        sh "cd build && mkdir ${env.BUILD_NUMBER} && cp ci/* ${env.BUILD_NUMBER} -r && cp chain33* Dockerfile* docker* *.sh ${env.BUILD_NUMBER}/ && cd ${env.BUILD_NUMBER}/ && ./docker-compose-pre.sh run ${env.BUILD_NUMBER} all "
                    }
                }
            }

            post {
                always {
                    dir("${PROJ_DIR}"){
                        sh "cd build/${env.BUILD_NUMBER} && ./docker-compose-pre.sh down ${env.BUILD_NUMBER} all && cd .. && rm -rf ${env.BUILD_NUMBER} && cd .. && make clean "
                    }
                }
            }
        }
    }

    post {
        always {
            echo 'One way or another, I have finished'
            // clean up our workspace
            deleteDir()
        }

        success {
            echo 'I succeeeded!'
            echo "email user: ${gitlabUserEmail}"
            mail to: "${gitlabUserEmail}",
                 subject: "Successed Pipeline: ${currentBuild.fullDisplayName}",
                 body: "this is success with ${env.BUILD_URL}"
        }

        failure {
            echo 'I failed '
            echo "email user: ${gitlabUserEmail}"
            mail to: "${gitlabUserEmail}",
                 subject: "Failed Pipeline: ${currentBuild.fullDisplayName}",
                 body: "Something is wrong with ${env.BUILD_URL}"
        }
    }
}
