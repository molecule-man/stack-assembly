Feature: stas sync with hooks

    Scenario: sync executes all the possible hooks
        Given file "cfg.yaml" exists:
            """
            hooks:
              pre:
                - ["sh", "-c", "echo root pre executed > hooks.log"]
              post:
                - ["sh", "-c", "echo root post executed >> hooks.log"]
            stacks:
              stack1:
                name: stastest-hooks-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
                hooks:
                  pre:
                    - ["sh", "-c", "echo stack pre executed >> hooks.log"]
                  post:
                    - ["sh", "-c", "echo stack post executed >> hooks.log"]
                  precreate:
                    - ["sh", "-c", "echo stack precreate executed >> hooks.log"]
                  postcreate:
                    - ["sh", "-c", "echo stack postcreate executed >> hooks.log"]
                  preupdate:
                    - ["sh", "-c", "echo stack preupdate executed >> hooks.log"]
                  postupdate:
                    - ["sh", "-c", "echo stack postupdate executed >> hooks.log"]
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                    ClusterName: !Ref AWS::StackName
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then file "hooks.log" should contain exactly:
            """
            root pre executed
            stack pre executed
            stack precreate executed
            stack postcreate executed
            stack post executed
            root post executed
            """
        When I modify file "tpls/stack1.yml":
            """
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                    ClusterName: !Sub "${AWS::StackName}-modify"
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        Then file "hooks.log" should contain exactly:
            """
            root pre executed
            stack pre executed
            stack preupdate executed
            stack postupdate executed
            stack post executed
            root post executed
            """
