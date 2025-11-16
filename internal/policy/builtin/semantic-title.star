"""hanles semantic commit title enforcement policy."""

default_pattern = r"^(feat|fix|docs|style|refactor|perf|test|chore)(\\(.+\\))?[:!] .{10,70}$"
default_message = None

def evaluate(context, pattern = default_pattern, message = default_message):
    """
    Evaluate semantic title policy.

    Args:
      context (dict): A dictionary containing 'principal', 'resource', and 'reviews'.
      pattern (str): Regex pattern that the title must match.
      message (str): Custom denial message if the pattern does not match.

    Returns:
      dict: A dictionary with 'decision', 'reason', and 'approval_requirements'.
    """

    matcher = re.compile(pattern)
    title = context["resource"].get("title", "")

    if matcher.match(title):
        return allow()

    if message == None:
        message = "Commit title '{}' does not match required pattern /{}/.".format(title, pattern)

    return deny(message = message)
