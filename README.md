# flak-analyzer

This Flak Analyzer is a project to summerize JUNIT test reports generated from GitHub workflow where the  results are
 uploaded to the repository at the end of every (failed) run. The analyzer generates reports by aggregating all the
  results availble and sort the failed tests by their occurrences from high to low. This analyzer is a
   standalone project that can be used as a part of the workflow to download test results from GitHub as artifacts and
    aggregate them.
    
To use this project, you need to set up the following:
1. Upload your test results to your repository via workflow.
2. Include this project in your workflow and supply filters accordingly (commit or Test Suite Name).
3. [Create a personal access token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token)
 to feed into the workflow for access your repository. (You need the token for downloading the artifacts)
4. Enjoy the anlysis report. 
    
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

## Analysis Report Example
```yaml
totaltestcount: 30
flaktestcount: 43
skippedtestcount: 4
flaktests:
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
