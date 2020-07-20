# Flake-Analyzer

This Flake Analyzer is a project to summarize JUNIT test reports generated from GitHub workflow where the results are
 uploaded to the repository at the end of every (failed) run. The analyzer generates reports by aggregating all the
  results available and sort the failed tests by their occurrences from high to low. This analyzer is a
   standalone project that can be used as a part of the workflow to download test results from GitHub as artifacts and
    aggregate them.

The commenter is a feature to post flake report to a pull request based on the failed test runs from that PR. The analyzer performs a periodic check against the repo
 and post reports if a new artifact that has the new run ID and has the commit id associated with the PR exists.

The flake analyzer supports periodic reports for different time windows, errors from specific PRs, and commits with respect to different tests.
The flake analyzer has the following modes:
  - Periodically generate report artifacts for a specific repository with respect to individual tests. (requires GITHUB_TOKEN)
  - Comment on PRs failed for specific tests with aggregated error report. (requires Personal Access Token with access to download REPO artifact and post comment)
    
To use the periodic flake analysis reporting feature, you need to set up the following:
1. Upload your test results to your repository via workflow using this [template](#github-upload-junit-test-result-as-artifacts)).
2. Include this project in your Repo's workflow and supply filters accordingly including Test Suite Name supplied as
 part of your uploaded artifact (eg. for [example](#report-daily/weekly-flake-failures)).
3. Enjoy the analysis report as artifacts. 

To use the commenting report feature, you need to set up the following:
1. Upload your test results to your repository via workflow using this [template
](#github-upload-junit-test-result-as-artifacts)).
2. Create a new workflow on this repo to monitor your repositories using this [template](#enable-commenter).
3. [Create a personal access token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) and add to flake-analyzer Repo
 to feed into the workflow from the second step for access your repository. (You need the token for downloading the artifacts)
4. Test out commenting feature by opening a PR from non-fork repository.(forked repository can not access token)

## Github Upload JUNIT Test Result as Artifacts

To analyize test results, you need to upload your test results onto Github. The following is an example of the
 worflow configuration `.github/workflows/<Your Test Config>.yml`. See https://github.com/actions/upload-artifact for
  more information.
  
  The important thing is to upload your test artifacts in the format <Test Suite Name\>-<Commit\>-<Run ID\>, which is
   used in the example below.
```yaml
<Your Reo>/.github/workflows/<your test>.yml
name: <Test Name>
on:
  pull_request:
jobs:
  <Job Name>:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - run: <Your Test>
      - name: Archive production artifacts # test results are only uploaded if any of the e2e tests fails
        if: ${{ failure() }}
        uses: actions/upload-artifact@v2
        with:
          name: <Test Suite Name>-${{(github.event.pull_request.head.sha||github.sha)}}-${{ github.run_id }}
          path: <Path/To/Your/Artifacts>
```

## Report Daily/Weekly Flake Failures

```yaml
<Your Reo>/.github/workflows/<your periodics>.yml
name: flake-analyzer-periodics
on:
  schedule:
    - cron: '0 1 * * *' # daily
jobs:
  generate-flake-analysis-report:
    runs-on: ubuntu-latest
    steps:
      - name: Periodic Flake Report
        env:
          OWNER: <your repo owner>
          REPO: <your repo>
          TEST_SUITE: <Test Suite Name>
          TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          git clone -b v0.1.1 https://github.com/operator-framework/flake-analyzer.git
          cd ./flake-analyzer
          make report-today  OUTPUT_FILE=./report/artifacts/flake-report-today-$(date +"%m-%d-%Y").yaml
          make report-last-7-days OUTPUT_FILE=./report/artifacts/flake-report-last-7-days-$(date +"%m-%d-%Y").yaml
          make report-prev-7-days OUTPUT_FILE=./report/artifacts/flake-report-prev-7-days-$(date +"%m-%d-%Y").yaml
      - name: Archive Reoport artifacts # test results are only uploaded if any of the e2e tests fails
        uses: actions/upload-artifact@v2
        with:
          name: flake-report-${{ github.run_id }}
          path: ${{ github.workspace }}/flake-analyzer/report/artifacts/*
```

## Enable Commenter

```yaml
<flake-analyzer>/.github/workflows/<your commenter>.yml
name: Post Report As Comment On PR
on:
  pull_request:
  schedule:
    - cron: '*/15 * * * *'
jobs:
  post-report-as-pr-comment:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Comment On PR
        env:
          OWNER: <your repo owner>
          REPO: <your repo>
          LOWNER: <Owner of flake-analyzer, e.g. operator-framework>
          LREPO: flake-analyzer
          TEST_SUITE: <your test suite>
          PROGRESS_FILE: ./artifacts/commenter-progress-<Your Bot Name>.yaml
          ARTIFACT: flake-bot-<your owner>-artifact
          TOKEN: ${{secrets.<Personal Access Token Name>}}
        run: make commenter
      - uses: actions/upload-artifact@v2 # upload your commenter progress
        with:
          name: flake-bot-<your owner>-artifact
          path: ./artifacts/commenter-progress-<Your Bot Name>.yaml
```

## Analysis Report Example
```yaml
totaltestcount: 30
flaketestcount: 43
skippedtestcount: 4
flaketests:
- classname: End-to-end
  name: Installing bundles with new object types when a bundle with a pdb, priorityclass,
    and VPA object is installed should create the additional bundle objects
  counts: 28
  details:
  - count: 28
    error:
      type: Failure
      body: |-
        /home/runner/work/operator-lifecycle-manager/operator-lifecycle-manager/test/e2e/bundle_e2e_test.go:78
        Timed out after 60.000s.
        expected no error getting pdb object associated with CSV
        Expected success, but got an error:
            <*errors.StatusError | 0xc00113eaa0>: {
                ErrStatus: {
                    TypeMeta: {Kind: "", APIVersion: ""},
                    ListMeta: {
                        SelfLink: "",
                        ResourceVersion: "",
                        Continue: "",
                        RemainingItemCount: nil,
                    },
                    Status: "Failure",
                    Message: "poddisruptionbudgets.policy \"busybox-pdb\" not found",
                    Reason: "NotFound",
                    Details: {
                        Name: "busybox-pdb",
                        Group: "policy",
                        Kind: "poddisruptionbudgets",
                        UID: "",
                        Causes: nil,
                        RetryAfterSeconds: 0,
                    },
                    Code: 404,
                },
            }
            poddisruptionbudgets.policy "busybox-pdb" not found
        /home/runner/work/operator-lifecycle-manager/operator-lifecycle-manager/test/e2e/bundle_e2e_test.go:98
    systemout: |
      15:13:31.0741: UpgradePending (busybox.v2.0.0): &ObjectReference{Kind:InstallPlan,Namespace:operators,Name:install-cgghb,UID:580aae23-1784-4f6b-9486-ce9479f563c6,APIVersion:operators.coreos.com/v1alpha1,ResourceVersion:5826,FieldPath:,}
      15:13:31.731: UpgradePending (busybox.v2.0.0): &ObjectReference{Kind:InstallPlan,Namespace:operators,Name:install-cgghb,UID:580aae23-1784-4f6b-9486-ce9479f563c6,APIVersion:operators.coreos.com/v1alpha1,ResourceVersion:5826,FieldPath:,}
      15:13:32.7313: UpgradePending (busybox.v2.0.0): &ObjectReference{Kind:InstallPlan,Namespace:operators,Name:install-cgghb,UID:580aae23-1784-4f6b-9486-ce9479f563c6,APIVersion:operators.coreos.com/v1alpha1,ResourceVersion:5826,FieldPath:,}
      15:13:33.7321: UpgradePending (busybox.v2.0.0): &ObjectReference{Kind:InstallPlan,Namespace:operators,Name:install-cgghb,UID:580aae23-1784-4f6b-9486-ce9479f563c6,APIVersion:operators.coreos.com/v1alpha1,ResourceVersion:5826,FieldPath:,}
      15:13:34.7309: UpgradePending (busybox.v2.0.0): &ObjectReference{Kind:InstallPlan,Namespace:operators,Name:install-cgghb,UID:580aae23-1784-4f6b-9486-ce9479f563c6,APIVersion:operators.coreos.com/v1alpha1,ResourceVersion:5826,FieldPath:,}
      15:13:35.7534: UpgradePending (busybox.v2.0.0): &ObjectReference{Kind:InstallPlan,Namespace:operators,Name:install-cgghb,UID:580aae23-1784-4f6b-9486-ce9479f563c6,APIVersion:operators.coreos.com/v1alpha1,ResourceVersion:5826,FieldPath:,}
      15:13:36.7354: AtLatestKnown (busybox.v2.0.0): &ObjectReference{Kind:InstallPlan,Namespace:operators,Name:install-cgghb,UID:580aae23-1784-4f6b-9486-ce9479f563c6,APIVersion:operators.coreos.com/v1alpha1,ResourceVersion:5826,FieldPath:,}
      skipping cleanup
    systemerr: ""
  commits:
  - 0b8233d0c2eefb9c3b7402f3709525c7ec6752a7
  - 15f0d9741dd33e2672b552540fa4ed564cec92ec
  - ...
  meandurationsec: 103.58520275000001
...

skippedtests:
- classname: End-to-end
  name: Subscriptions create required objects from Catalogs Given a Namespace when
    a CatalogSource is created with a bundle that contains prometheus objects creating
    a subscription using the CatalogSource should have created the expected prometheus
    objects
  counts: 30
  details: []
  commits:
  - 0b8233d0c2eefb9c3b7402f3709525c7ec6752a7
  - 15f0d9741dd33e2672b552540fa4ed564cec92ec
  - ...
  meandurationsec: 5.392025833333333
...
```
