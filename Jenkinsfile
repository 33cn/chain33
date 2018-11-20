
pipeline {
    agent any

    environment {
        GOPATH = "${WORKSPACE}"
        PROJ_DIR = "${WORKSPACE}/src/github.com/33cn/chain33"
    }

    options {
        timeout(time: 2,unit: 'HOURS')
        retry(1)
        timestamps()
        gitLabConnection('gitlab33')
        gitlabBuilds(builds: ['check'])
        checkoutToSubdirectory "src/github.com/33cn/chain33"
    }

    stages {
        stage('check') {
            steps {
                dir("${PROJ_DIR}"){
                    gitlabCommitStatus(name: 'check'){
                        sh "git branch"
                        sh "make auto_ci branch=${env.ghprbSourceBranch} originx=${env.ghprbAuthorRepoGitUrl}"
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
            echo "email user: ${ghprbActualCommitAuthorEmail}"
            mail to: "${ghprbActualCommitAuthorEmail}",
                 subject: "Successed Pipeline: ${currentBuild.fullDisplayName}",
                 body: "this is success with ${env.BUILD_URL}"
        }

        failure {
            echo 'I failed '
            echo "email user: ${ghprbActualCommitAuthorEmail}"
            mail to: "${ghprbActualCommitAuthorEmail}",
                 subject: "Failed Pipeline: ${currentBuild.fullDisplayName}",
                 body: "Something is wrong with ${env.BUILD_URL}"
        }
    }
}
