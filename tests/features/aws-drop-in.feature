Feature: drop-in replacement for aws cloudformation create-stack|update-stack|deploy

    @short
    Scenario: aws cloudformation create-stack
        Given file "tpls/cluster.yml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref "AWS::StackName"
            """
        When I successfully run "--no-interaction cloudformation create-stack --stack-name stastest-%scenarioid% --template-body file://tpls/cluster.yml --output text"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"
        And output should be exactly:
            """
            {%with StackInfo "stastest-%scenarioid%"%}{% .StackId %}{%end%}
            """

    @short
    Scenario: aws cloudformation create-stack json output
        Given file "tpls/cluster.yml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref "AWS::StackName"
            """
        When I successfully run "--no-interaction cloudformation create-stack --stack-name stastest-%scenarioid% --template-body file://tpls/cluster.yml --output json"
        Then stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"
        And output should be exactly:
            """
            {
                "StackId": "{%with StackInfo "stastest-%scenarioid%"%}{% .StackId %}{%end%}"
            }
            """

    
    Scenario: aws cloudformation update-stack
        Given file "tpls/cluster.yml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref "AWS::StackName"
            """
        Given file "tpls/cluster.v2.yml" exists:
            """
            Resources:
              Cluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "${AWS::StackName}-v2"
            """
        When I successfully run "--no-interaction cloudformation create-stack --stack-name stastest-%scenarioid% --template-body file://tpls/cluster.yml"
        And I successfully run "--no-interaction cloudformation update-stack --stack-name stastest-%scenarioid% --template-body file://tpls/cluster.v2.yml"
        Then stack "stastest-%scenarioid%" should have status "UPDATE_COMPLETE"
