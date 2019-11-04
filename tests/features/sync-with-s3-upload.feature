Feature: stas sync with s3 upload

    @mock
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
