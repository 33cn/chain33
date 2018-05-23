// ci server: http://47.97.200.227:8080
// user/pass: ci/ci

pipeline {
    agent any

    environment {
        GOPATH = "${WORKSPACE}"
        PROJ_DIR = "${WORKSPACE}/src/gitlab.33.cn/chain33/chain33"
    }

    options {
        timeout(time: 1,unit: 'HOURS')
        retry(1)
        buildDiscarder(logRotator(numToKeepStr: '1'))
        disableConcurrentBuilds()
        timestamps()
        gitLabConnection 'gitlab33'
        // gitlabBuilds(builds: ['build', 'test', 'deploy'])
        gitlabCommitStatus(name: 'jenkins')
        checkoutToSubdirectory 'src/gitlab.33.cn/chain33/chain33'
    }

    stages {
        stage('build') {
            steps { 
                dir("${PROJ_DIR}"){
                    sh 'make checkgofmt'
                    sh 'make linter' 
                    sh 'make build_ci' 
                }
            }
        }

        stage('test'){
            steps {
                dir("${PROJ_DIR}"){
                    sh 'cd build && docker-compose down && cd ..'
                    sh 'make test'
                    sh 'export CC=clang-5.0 && make msan'
                }
            }
        }

        stage('deploy') {
            steps {
                dir("${PROJ_DIR}"){
                    sh 'make build_ci'
                    sh 'make docker-compose'
                    sh 'cd build && docker-compose down && cd ..'
                }
            }
        }
    }

    post {
        always {
            echo 'One way or another, I have finished'
            // clean up our workspace
            // deleteDir()
        }

        success {
            echo 'I succeeeded!'
            updateGitlabCommitStatus name: 'build', state: 'success'
            // deleteDir()
            echo "email user: ${gitlabUserEmail}"
            mail to: "${gitlabUserEmail}",
                 subject: "Successed Pipeline: ${currentBuild.fullDisplayName}",
                 body: "this is success with ${env.BUILD_URL}"
        }

        unstable {
            echo 'I am unstable'
        }

        failure {
            echo 'I failed '
            updateGitlabCommitStatus name: 'build', state: 'failed'
            echo "email user: ${gitlabUserEmail}"
            mail to: "${gitlabUserEmail}",
                 subject: "Failed Pipeline: ${currentBuild.fullDisplayName}",
                 body: "Something is wrong with ${env.BUILD_URL}/console|Console Output"
        }

        changed {
            echo 'Things were different before...'
        }
    }
}
