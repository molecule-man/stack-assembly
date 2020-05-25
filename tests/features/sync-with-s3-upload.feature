Feature: stas sync with s3 upload

    @nomock @fix-in-mock
    Scenario: sync single valid template with enforced s3 upload
        Given file "cfg.yaml" exists:
            """
            settings:
              s3Settings:
                bucketName: stastest-%featureid%-1
                thresholdSize: 5
            stacks:
              stack1:
                name: stastest-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'

            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest-%scenarioid%
            """
        And file "tpls/bucket.yml" exists:
            """
            Resources:
              Bucket:
                Type: AWS::S3::Bucket
                Properties:
                  BucketName: stastest-%featureid%-1
                  Tags:
                    - Key: STAS_TEST
                      Value: true
            """
        When I successfully run "deploy --no-interaction stastest-bucket-%scenarioid% tpls/bucket.yml"
        And I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    @nomock @fix-in-mock
    Scenario: sync stack with body over 51200 without specifying bucket
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-%scenarioid%
                path: tpls/stack.yml
                tags:
                  STAS_TEST: '%featureid%'

            """
        And file "tpls/stack.yml" exists:
            """
            Resources:
              MySecret:
                Type: 'AWS::SecretsManager::Secret'
                Properties:
                  SecretString: '{"whatever":"%longstring%"}'
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"
