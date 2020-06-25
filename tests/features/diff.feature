Feature: stas diff

    @short
    Scenario: diff two stacks one of which is changed
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-diff1-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
              stack2:
                name: stastest-diff2-%scenarioid%
                path: tpls/stack2.yml
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
        And file "tpls/stack2.yml" exists:
            """
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: stastest2-%scenarioid%
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
        And I successfully run "diff -c cfg.yaml --nocolor"
        Then output should be exactly:
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
