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

    @short
    Scenario: diff stack with no real changes (only yaml stylistic changes)
        Given file "cfg.yaml" exists:
            """
            name: stastest-no-change-%scenarioid%
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
                  ClusterName: !Ref AWS::StackName
                  Tags:
                    - Key: SOME_TAG
                      Value: some long string that can take two lines
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        When I modify file "tpls/stack1.yml":
            """
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref AWS::StackName
                  Tags:
                    - Key: SOME_TAG
                      Value: some long string that
                        can take two lines
            """
        And I successfully run "diff -c cfg.yaml --nocolor"
        Then output should be exactly:
            """
            """

    @short
    Scenario: diff json stack with yaml stack
        Given file "cfg.yaml" exists:
            """
            name: stastest-json-yaml-diff-%scenarioid%
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
                  ClusterName: stastest-json-yaml-diff-%scenarioid%
                  Tags:
                    - Key: SOME_TAG
                      Value: some value
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        When I modify file "tpls/stack1.yml":
            """
            {
              "Resources": {
                "EcsCluster": {
                  "Type": "AWS::ECS::Cluster",
                  "Properties": {
                    "ClusterName": "stastest-json-yaml-diff-%scenarioid%",
                    "Tags": [{
                      "Key": "SOME_TAG",
                      "Value": "some other value"
                    }]
                  }
                }
              }
            }
            """
        And I successfully run "diff -c cfg.yaml --nocolor"
        Then output should be exactly:
            """
            --- old/stastest-json-yaml-diff-%scenarioid%
            +++ new/stastest-json-yaml-diff-%scenarioid%
            @@ -4,11 +4,11 @@
                   "Properties": {
                     "ClusterName": "stastest-json-yaml-diff-%scenarioid%",
                     "Tags": [
                       {
                         "Key": "SOME_TAG",
            -            "Value": "some value"
            +            "Value": "some other value"
                       }
                     ]
                   },
                   "Type": "AWS::ECS::Cluster"
                 }
            """
