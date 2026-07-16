export type ThreadStartedEvent = {
  type: "thread.started";
  thread_id: string;
};

export type ThreadFailedEvent = {
  type: "thread.failed";
  message: string;
};

export type ThreadEvent = ThreadStartedEvent | ThreadFailedEvent;
