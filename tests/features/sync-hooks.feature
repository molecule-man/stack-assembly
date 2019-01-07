Feature: stas sync with hooks

    @wip
    Scenario: sync executes all the possible hooks
        Given file "cfg.yaml" exists:
            """
            hooks:
              pre:
                - ["sh", "-c", "echo root pre executed > hooks.log"]
              post:
                - ["sh", "-c", "echo root post executed >> hooks.log"]
              presync:
                - ["sh", "-c", "echo root presync executed >> hooks.log"]
              postsync:
                - ["sh", "-c", "echo root postsync executed >> hooks.log"]
              precreate:
                - ["sh", "-c", "echo root precreate executed >> hooks.log"]
              postcreate:
                - ["sh", "-c", "echo root postcreate executed >> hooks.log"]
              preupdate:
                - ["sh", "-c", "echo root preupdate executed >> hooks.log"]
              postupdate:
                - ["sh", "-c", "echo root postupdate executed >> hooks.log"]
            stacks:
              stack1:
                name: stastest-hooks-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
                hooks:
                  presync:
                    - ["sh", "-c", "echo stack presync executed >> hooks.log"]
                  postsync:
                    - ["sh", "-c", "echo stack postsync executed >> hooks.log"]
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
            root presync executed
            stack presync executed
            root precreate executed
            stack precreate executed
            root postsync executed
            stack postsync executed
            root postcreate executed
            stack postcreate executed
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
            root presync executed
            stack presync executed
            root preupdate executed
            stack preupdate executed
            root postsync executed
            stack postsync executed
            root postupdate executed
            stack postupdate executed
            root post executed
            """
