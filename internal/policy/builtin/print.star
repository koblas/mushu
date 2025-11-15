"""
Debugging policy that prints information during evaluation.
"""

def evaluate(context, **kwargs):
    """
    Evaluates the debug policy by printing context and keyword arguments.

    Args:
        context: The evaluation context.
        **kwargs: Additional keyword arguments.

    Returns:
        The result of the policy evaluation.
    """
    print("Debug Policy Evaluation")
    print("Context:", context)
    for key, value in kwargs.items():
        print(key, ":", value)

    return allow()
