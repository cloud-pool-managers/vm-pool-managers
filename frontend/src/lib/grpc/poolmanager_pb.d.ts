import * as jspb from 'google-protobuf'

import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb'; // proto import: "google/protobuf/empty.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class RessourceRequest extends jspb.Message {
  getUser(): string;
  setUser(value: string): RessourceRequest;

  getDataMap(): jspb.Map<string, string>;
  clearDataMap(): RessourceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RessourceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RessourceRequest): RessourceRequest.AsObject;
  static serializeBinaryToWriter(message: RessourceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RessourceRequest;
  static deserializeBinaryFromReader(message: RessourceRequest, reader: jspb.BinaryReader): RessourceRequest;
}

export namespace RessourceRequest {
  export type AsObject = {
    user: string;
    dataMap: Array<[string, string]>;
  };
}

export class UserRequest extends jspb.Message {
  getUser(): string;
  setUser(value: string): UserRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UserRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UserRequest): UserRequest.AsObject;
  static serializeBinaryToWriter(message: UserRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UserRequest;
  static deserializeBinaryFromReader(message: UserRequest, reader: jspb.BinaryReader): UserRequest;
}

export namespace UserRequest {
  export type AsObject = {
    user: string;
  };
}

export class StreamRessourceResponse extends jspb.Message {
  getUser(): string;
  setUser(value: string): StreamRessourceResponse;

  getStatus(): Status;
  setStatus(value: Status): StreamRessourceResponse;

  getType(): Type;
  setType(value: Type): StreamRessourceResponse;

  getDataMap(): jspb.Map<string, string>;
  clearDataMap(): StreamRessourceResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StreamRessourceResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StreamRessourceResponse): StreamRessourceResponse.AsObject;
  static serializeBinaryToWriter(message: StreamRessourceResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StreamRessourceResponse;
  static deserializeBinaryFromReader(message: StreamRessourceResponse, reader: jspb.BinaryReader): StreamRessourceResponse;
}

export namespace StreamRessourceResponse {
  export type AsObject = {
    user: string;
    status: Status;
    type: Type;
    dataMap: Array<[string, string]>;
  };
}

export class Image extends jspb.Message {
  getId(): string;
  setId(value: string): Image;

  getName(): string;
  setName(value: string): Image;

  getStatus(): string;
  setStatus(value: string): Image;

  getTags(): string;
  setTags(value: string): Image;

  getContainerFormat(): string;
  setContainerFormat(value: string): Image;

  getDiskFormat(): string;
  setDiskFormat(value: string): Image;

  getMinDiskGigabytes(): number;
  setMinDiskGigabytes(value: number): Image;

  getMinRamMegabytes(): number;
  setMinRamMegabytes(value: number): Image;

  getOwner(): string;
  setOwner(value: string): Image;

  getProtected(): boolean;
  setProtected(value: boolean): Image;

  getVisibility(): string;
  setVisibility(value: string): Image;

  getHidden(): boolean;
  setHidden(value: boolean): Image;

  getChecksum(): string;
  setChecksum(value: string): Image;

  getSizeBytes(): number;
  setSizeBytes(value: number): Image;

  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Image;
  hasCreatedAt(): boolean;
  clearCreatedAt(): Image;

  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Image;
  hasUpdatedAt(): boolean;
  clearUpdatedAt(): Image;

  getFile(): string;
  setFile(value: string): Image;

  getSchema(): string;
  setSchema(value: string): Image;

  getVirtualSize(): number;
  setVirtualSize(value: number): Image;

  getImportMethods(): string;
  setImportMethods(value: string): Image;

  getStoreIds(): string;
  setStoreIds(value: string): Image;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Image.AsObject;
  static toObject(includeInstance: boolean, msg: Image): Image.AsObject;
  static serializeBinaryToWriter(message: Image, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Image;
  static deserializeBinaryFromReader(message: Image, reader: jspb.BinaryReader): Image;
}

export namespace Image {
  export type AsObject = {
    id: string;
    name: string;
    status: string;
    tags: string;
    containerFormat: string;
    diskFormat: string;
    minDiskGigabytes: number;
    minRamMegabytes: number;
    owner: string;
    pb_protected: boolean;
    visibility: string;
    hidden: boolean;
    checksum: string;
    sizeBytes: number;
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    file: string;
    schema: string;
    virtualSize: number;
    importMethods: string;
    storeIds: string;
  };
}

export class Flavor extends jspb.Message {
  getId(): string;
  setId(value: string): Flavor;

  getName(): string;
  setName(value: string): Flavor;

  getDisk(): number;
  setDisk(value: number): Flavor;

  getRam(): number;
  setRam(value: number): Flavor;

  getVcpus(): number;
  setVcpus(value: number): Flavor;

  getRxtxFactor(): number;
  setRxtxFactor(value: number): Flavor;

  getSwap(): number;
  setSwap(value: number): Flavor;

  getEphemeral(): number;
  setEphemeral(value: number): Flavor;

  getIsPublic(): boolean;
  setIsPublic(value: boolean): Flavor;

  getDescription(): string;
  setDescription(value: string): Flavor;

  getExtraSpecs(): string;
  setExtraSpecs(value: string): Flavor;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Flavor.AsObject;
  static toObject(includeInstance: boolean, msg: Flavor): Flavor.AsObject;
  static serializeBinaryToWriter(message: Flavor, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Flavor;
  static deserializeBinaryFromReader(message: Flavor, reader: jspb.BinaryReader): Flavor;
}

export namespace Flavor {
  export type AsObject = {
    id: string;
    name: string;
    disk: number;
    ram: number;
    vcpus: number;
    rxtxFactor: number;
    swap: number;
    ephemeral: number;
    isPublic: boolean;
    description: string;
    extraSpecs: string;
  };
}

export class Network extends jspb.Message {
  getId(): string;
  setId(value: string): Network;

  getName(): string;
  setName(value: string): Network;

  getDescription(): string;
  setDescription(value: string): Network;

  getAdminStateUp(): boolean;
  setAdminStateUp(value: boolean): Network;

  getStatus(): string;
  setStatus(value: string): Network;

  getTenantId(): string;
  setTenantId(value: string): Network;

  getProjectId(): string;
  setProjectId(value: string): Network;

  getShared(): boolean;
  setShared(value: boolean): Network;

  getRevisionNumber(): number;
  setRevisionNumber(value: number): Network;

  getSubnets(): string;
  setSubnets(value: string): Network;

  getAvailabilityZoneHints(): string;
  setAvailabilityZoneHints(value: string): Network;

  getTags(): string;
  setTags(value: string): Network;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Network.AsObject;
  static toObject(includeInstance: boolean, msg: Network): Network.AsObject;
  static serializeBinaryToWriter(message: Network, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Network;
  static deserializeBinaryFromReader(message: Network, reader: jspb.BinaryReader): Network;
}

export namespace Network {
  export type AsObject = {
    id: string;
    name: string;
    description: string;
    adminStateUp: boolean;
    status: string;
    tenantId: string;
    projectId: string;
    shared: boolean;
    revisionNumber: number;
    subnets: string;
    availabilityZoneHints: string;
    tags: string;
  };
}

export enum Status {
  STATUS_UNKNOWN = 0,
  CREATE = 1,
  UPDATE = 2,
  DELETE = 3,
}
export enum Type {
  TYPE_UNKNOWN = 0,
  SERVERPOOL = 1,
  SERVER = 2,
  CONFIG = 3,
}
