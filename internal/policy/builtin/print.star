"""
Debugging policy that prints information during evaluation.
"""

def evaluate(context, **kwargs):
    print("Debug Policy Evaluation")
    print("Context:", context)
    for key, value in kwargs.items():
        print(key, ":", value)

    return allow()
