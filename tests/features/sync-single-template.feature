Feature: stas deploy only single template

    @short @mock
    Scenario: deploy single template
        Given file "tpl.yaml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref AWS::StackName
            """
        When I successfully run "deploy --no-interaction stastest-%scenarioid% tpl.yaml"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    @short @mock
    Scenario: deploy single template with parameters
        Given file "tpl.yaml" exists:
            """
            Parameters:
              version:
                Type: String
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "${AWS::StackName}-${version}"
            """
        When I successfully run "deploy --no-interaction stastest-%scenarioid% tpl.yaml -v version=1-2-3"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"
