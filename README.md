# flak-analyzer

This Flak Analyzer is a project to summerize JUNIT test reports generated from GitHub workflow where the  results are
 uploaded to the repository at the end of every (failed) run. The analyzer generates reports by aggregating all the
  results availble and sort the failed tests by their occurrence from highest to lowest. This analyzer is a
   standalone project that can be used as part of the workflow to download test results from GitHub as artifacts and
    aggregate them.
    
To use this project, you need to set up the following:
1. Upload your test results to your repository via workflow.
2. Include this project in your workflow and supply filters accordingly (commit or Test Suite Name).
3. [Create a personal access token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token)
 to feed into the workflow for access your repository. (You need the token for downloading the artifacts)
4. Enjoy the anlysis via std_out and report artifacts. 
    
## Github Upload JUNIT Test Result as Artifacts

To analyize test results, you need to upload your test results onto Github. The following is an example of the
 worflow configuration `.github/workflows/<Your Test Config>.yml`. See https://github.com/actions/upload-artifact for
  more information.
  
  The important thing is to upload your test artifacts in the format <Test Suite Name\>-<Commit\>-<Run ID\>, which is
   used in the example below.
```yaml
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
          name: <Test Suite Name>-${{ github.sha }}-${{ github.run_id }}
          path: <Path/To/Your/Artifacts>
```
 