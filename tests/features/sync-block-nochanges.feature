Feature: stas sync block stack when there are no changes

    @nomock
    Scenario: blocking happens even if there are no changes in body
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-block-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              EcsCluster1:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest-%scenarioid%
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        And I modify file "cfg.yaml":
            """
            stacks:
              stack1:
                name: stastest-block-%scenarioid%
                path: tpls/stack1.yml
                blocked:
                  - EcsCluster1
                tags:
                  STAS_TEST: '%featureid%'
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        Then output should contain:
            """
            No changes to be synchronized
            """
        And output should contain:
            """
            Blocking resource EcsCluster1
            """
        When I modify file "tpls/stack1.yml":
            """
            Resources:
              EcsCluster1:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest-mod-%scenarioid%
            """
        And I run "sync -c cfg.yaml --no-interaction"
        Then exit code should not be zero
        And output should contain:
            """
            does not allow [Update:Replace, Update:Delete]
            """
        And stack "stastest-block-%scenarioid%" should have status "UPDATE_ROLLBACK_COMPLETE"
