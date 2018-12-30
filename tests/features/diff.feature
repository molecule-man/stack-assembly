Feature: stas diff

    @wip
    Scenario: diff
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-diff1-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest1-%scenarioid%
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        When I modify file "tpls/stack1.yml":
            """
            Resources:
              EcsCluster1:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest1-mod-%scenarioid%
            """
        And I successfully run "diff -c cfg.yaml"
        Then output should contain:
            """
            --- old/stastest-diff1-%scenarioid%
            +++ new/stastest-diff1-%scenarioid%
            @@ -1,5 +1,5 @@
             Resources:
            -  EcsCluster:
            +  EcsCluster1:
                 Type: AWS::ECS::Cluster
                 Properties:
            -      ClusterName: stastest1-%scenarioid%
            +      ClusterName: stastest1-mod-%scenarioid%
            """
