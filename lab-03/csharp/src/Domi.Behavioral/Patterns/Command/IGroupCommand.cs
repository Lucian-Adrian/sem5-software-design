using Domi.Behavioral.Patterns.Observer;

namespace Domi.Behavioral.Patterns.Command;

// ===========================================================================
// COMMAND
// Command:          IGroupCommand
// ConcreteCommands: JoinGroupCommand, CreateGroupCommand
// Invoker:          GroupCommandInvoker (queue + LIFO undo stack)
// Receiver:         IGroupWriter (Proxy -> Decorators -> GroupService)
// Business rules stay in the receiver and domain, never inside the command.
// ===========================================================================

/// <summary>COMMAND: encapsulates a group operation and how to undo it.</summary>
public interface IGroupCommand
{
    string Description { get; }
    GroupActivityEvent Execute();
    GroupActivityEvent Undo();
}
