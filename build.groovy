@Library(['edp-library-stages', 'edp-library-pipelines']) _

import hudson.FilePath
import groovy.json.*


PIPELINES_PATH_DEFAULT = "pipelines"

vars = [:]
vars['artifact'] = [:]
commonLib = null

node("master") {
    vars['pipelinesPath'] = env.PIPELINES_PATH ? PIPELINES_PATH : PIPELINES_PATH_DEFAULT

    def workspace = "${WORKSPACE.replaceAll("@.*", "")}@script"
    dir("${workspace}") {
        stash name: 'data', includes: "${vars.pipelinesPath}/**", useDefaultExcludes: false
        commonLib = load "${vars.pipelinesPath}/libs/common.groovy"
    }
}

node("ansible-slave") {
    stage("INITIALIZATION") {
        commonLib.getConstants(vars)
        try {
            dir("${vars.devopsRoot}") {
                unstash 'data'
            }
        } catch (Exception ex) {
            commonLib.failJob("[JENKINS][ERROR] Devops repository unstash has failed. Reason - ${ex}")
        }

        vars.application = [:]
        vars.application.name = vars.gerritProject
        vars.workDir = "/tmp/go/src/${vars.application.name}"
        vars.branch = env.GERRIT_BRANCH ? GERRIT_BRANCH : env.BRANCH

        currentBuild.displayName = "${currentBuild.number}-${vars.branch}"
        currentBuild.description = "Branch: ${vars.branch}"
        commonLib.getDebugInfo(vars)
    }

    dir("${vars.devopsRoot}/${vars.pipelinesPath}/stages/") {
        stage("CHECKOUT") {
            stage = load "git-checkout.groovy"
            stage.run(vars)

            def versionFile = new FilePath(Jenkins.getInstance().getComputer(env['NODE_NAME']).getChannel(), "${vars.workDir}/version.json").readToString()
            vars.application.version = "${new JsonSlurperClassic().parseText(versionFile).get(vars.application.name)}-${BUILD_NUMBER}"
            currentBuild.displayName = "${vars.application.version}"
        }

        stage("BUILD") {
            stage = load "operators/build.groovy"
            stage.run(vars)
        }

        stage("TESTS") {
            stage = load "operators/unit-tests.groovy"
            stage.run(vars)
        }

        stage("BUILD IMAGE") {
            stage = load "operators/build-image.groovy"
            stage.run(vars)
        }

        stage("PUSH-TO-NEXUS") {
            dir("${vars.workDir}") {
                sh "tar -cf ${vars.application.name}.tar deploy/*"
            }

            vars['artifact']['repository'] = "${vars.nexusRepository}-snapshots"
            vars['artifact']['version'] = vars.application.version
            vars['artifact']['id'] = vars.application.name
            vars['artifact']['path'] = "${vars.workDir}/${vars.application.name}.tar"
            stage = load "push-single-artifact-to-nexus.groovy"
            stage.run(vars)
        }

        stage("GIT-TAG") {
            vars['gitTag'] = vars.application.version
            stage = load "git-tag.groovy"
            stage.run(vars)
        }

        build job: "Deploy-${vars.application.name}", wait: false, parameters: [
                string(name: 'APPLICATION', value: "${vars.application.name}"),
                string(name: 'BRANCH', value: "${vars.branch}"),
                string(name: 'VERSION', value: "${vars.application.version}"),
        ]
    }
}

Build()