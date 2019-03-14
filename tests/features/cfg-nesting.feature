Feature: nested

    Scenario: nested stacks (1 level)
        Given file "cfg.yaml" exists:
            """
            stacks:
              nested_stack:
                stacks:
                  stack1:
                    name: stastest-1-%scenarioid%
                    path: tpls/stack1.yml
                    tags:
                      STAS_TEST: '%featureid%'
                  stack2:
                    name: stastest-2-%scenarioid%
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
                        ClusterName: !Ref AWS::StackName
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-1-%scenarioid%" should have status "CREATE_COMPLETE"
        And stack "stastest-2-%scenarioid%" should have status "CREATE_COMPLETE"

    Scenario: nested stacks (2 levels)
        Given file "cfg.yaml" exists:
            """
            stacks:
              nested_stack:
                stacks:
                  stack1:
                    name: stastest-1-%scenarioid%
                    path: tpls/stack1.yml
                    tags:
                      STAS_TEST: '%featureid%'
                    stacks:
                      stack2:
                        name: stastest-2-%scenarioid%
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
                        ClusterName: !Ref AWS::StackName
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-1-%scenarioid%" should have status "CREATE_COMPLETE"
        And stack "stastest-2-%scenarioid%" should have status "CREATE_COMPLETE"

    Scenario: stack on root level
        Given file "cfg.yaml" exists:
            """
            name: stastest-1-%scenarioid%
            path: tpls/stack1.yml
            tags:
              STAS_TEST: '%featureid%'
            stacks:
              stack2:
                name: stastest-2-%scenarioid%
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
                        ClusterName: !Ref AWS::StackName
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-1-%scenarioid%" should have status "CREATE_COMPLETE"
        And stack "stastest-2-%scenarioid%" should have status "CREATE_COMPLETE"

    @short
    Scenario: aws settings are propagated down the tree
        Given file "cfg.yaml" exists:
            """
            parameters:
              region: "{{ .AWS.Region }}"
            stacks:
              stack1:
                name: stastest-1-%scenarioid%
                path: tpls/stack.yml
                settings:
                  aws:
                    region: us-east-1
                parameters:
                  region: "{{ .AWS.Region }}"
                stacks:
                  stack2:
                    name: stastest-2-%scenarioid%
                    path: tpls/stack.yml
                    settings:
                      aws:
                        endpoint: www.example.com
                    parameters:
                      region: "{{ .AWS.Region }}"
            """
        And file "tpls/stack.yml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref AWS::StackName
            """
        When I successfully run "dump-config -c cfg.yaml -r eu-west-1 -f json"
        Then node "Settings.Aws" in json output should be:
            """
            {
              "Region": "eu-west-1",
              "Profile": "%aws_profile%",
              "Endpoint": ""
            }
            """
        And node "Parameters.region" in json output should be:
            """
            "eu-west-1"
            """
        And node "Stacks.stack1.Settings.Aws" in json output should be:
            """
            {
              "Region": "us-east-1",
              "Profile": "%aws_profile%",
              "Endpoint": ""
            }
            """
        And node "Stacks.stack1.Parameters.region" in json output should be:
            """
            "us-east-1"
            """
        And node "Stacks.stack1.Stacks.stack2.Settings.Aws" in json output should be:
            """
            {
              "Region": "us-east-1",
              "Profile": "%aws_profile%",
              "Endpoint": "www.example.com"
            }
            """
        And node "Stacks.stack1.Stacks.stack2.Parameters.region" in json output should be:
            """
            "us-east-1"
            """

    Scenario: executing specific nested stack
        Given file "cfg.yaml" exists:
            """
            stacks:
              staging:
                name: stastest-staging-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
                stacks:
                  app:
                    name: stastest-app-%scenarioid%
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
                  ClusterName: !Ref AWS::StackName
            """
        When I successfully run "sync -c cfg.yaml --no-interaction staging app"
        Then stack "stastest-app-%scenarioid%" should have status "CREATE_COMPLETE"
        But stack "stastest-staging-%scenarioid%" should not exist

    Scenario: I delete nested stack
        Given file "cfg.yaml" exists:
            """
            name: stastest-root-%scenarioid%
            path: tpls/stack1.yml
            tags:
              STAS_TEST: '%featureid%'
            stacks:
              parent:
                stacks:
                  child:
                    name: stastest-child-%scenarioid%
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
                  ClusterName: !Ref AWS::StackName
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        And stack "stastest-root-%scenarioid%" should have status "CREATE_COMPLETE"
        And stack "stastest-child-%scenarioid%" should have status "CREATE_COMPLETE"
        When I successfully run "delete -c cfg.yaml --no-interaction"
        Then stack "stastest-root-%scenarioid%" should not exist
        And stack "stastest-child-%scenarioid%" should not exist
