Feature: stas sync failure handling

    @short
    Scenario: sync fails on the stage of change set creation
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-fail1-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              "Fn::Transform":
                - Name: 'AWS::Include'
                  Parameters:
                    Location: 's3://non-existent-bucket-%scenarioid%/non-existent-tpl.yml'
            """
        When I run "sync -c cfg.yaml --no-interaction"
        Then exit code should not be zero
        And error contains:
            """
            ResourceNotReady: failed waiting for successful resource state. Status: FAILED, StatusReason: Transform AWS::Include failed with: S3 bucket [non-existent-bucket-%scenarioid%] does not exist.
            """
        And stack "stastest-fail1-%scenarioid%" should have status "REVIEW_IN_PROGRESS"
