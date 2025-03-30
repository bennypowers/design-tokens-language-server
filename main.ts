import { Server } from "./server.ts";

interface Message {
  id: number;
}

export interface RequestMessage extends Message {
  method: "initialize";
}

export interface ResponseMessage extends Message {
  method: "initialize";
}

Server.serve();
