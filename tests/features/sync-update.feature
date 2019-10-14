Feature: stas sync update

    @mock
    Scenario: sync create and then update
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-up-%scenarioid%
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
        And I modify file "tpls/stack1.yml":
            """
            Resources:
              EcsCluster1:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest-mod-%scenarioid%
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-up-%scenarioid%" should have status "UPDATE_COMPLETE"
