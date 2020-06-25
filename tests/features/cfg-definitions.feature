Feature: reuse (definitions)

    @short
    Scenario: reusing stacks
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                "$basedOn": stack_def1
                path: tpls/stack1.yml
            definitions:
              stack_def1:
                name: stastest-%scenarioid%
                path: whatever
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
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    @short
    Scenario: the stack in resulting config is based on referenced stack definition
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                "$basedOn": stack_def1
                path: tpls/stack1.yml
            definitions:
              stack_def1:
                name: stastest-%scenarioid%
                path: whatever
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
        When I successfully run "dump-config -c cfg.yaml --no-interaction --format json"
        Then node "Stacks.stack1.Path" in json output should be:
            """
            "tpls/stack1.yml"
            """
        Then node "Stacks.stack1.Name" in json output should be:
            """
            "stastest-%scenarioid%"
            """
        Then node "Stacks.stack1.Tags" in json output should be:
            """
            {
              "STAS_TEST": "%featureid%"
            }
            """
