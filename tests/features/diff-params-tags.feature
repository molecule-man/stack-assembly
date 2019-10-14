Feature: stas diff tags and parameters

    @short @mock
    Scenario: diff stack where body, parameters and tags are changed
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-diff1-%scenarioid%
                path: tpls/stack1.yml
                parameters:
                  env: dev
                  nameprefix: stastest
                tags:
                  STAS_TEST: "%featureid%"
            """
        And file "tpls/stack1.yml" exists:
            """
            Parameters:
              env:
                Type: String
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "{{ .Params.nameprefix }}-${env}-%scenarioid%"
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        When I modify file "cfg.yaml":
            """
            stacks:
              stack1:
                name: stastest-diff1-%scenarioid%
                path: tpls/stack1.yml
                parameters:
                  env: prod
                  nameprefix: stastest-mod
                tags:
                  STAS_TEST: "%featureid%"
                  NEW_TAG: "newtag"
            """
        And I successfully run "diff -c cfg.yaml --nocolor"
        Then output should be exactly:
            """
            --- old-parameters/stastest-diff1-%scenarioid%
            +++ new-parameters/stastest-diff1-%scenarioid%
            @@ -1 +1 @@
            -env: dev
            +env: prod

            --- old-tags/stastest-diff1-%scenarioid%
            +++ new-tags/stastest-diff1-%scenarioid%
            @@ -1 +1,2 @@
            +NEW_TAG: newtag
             STAS_TEST: %featureid%

            --- old/stastest-diff1-%scenarioid%
            +++ new/stastest-diff1-%scenarioid%
            @@ -3,6 +3,6 @@
                 Type: String
             Resources:
               EcsCluster:
                 Type: AWS::ECS::Cluster
                 Properties:
            -      ClusterName: !Sub "stastest-${env}-%scenarioid%"
            +      ClusterName: !Sub "stastest-mod-${env}-%scenarioid%"
            """
