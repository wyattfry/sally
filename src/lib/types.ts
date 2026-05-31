export type CapturedPage = {
  title: string;
  url: string;
  visibleText: string;
  mainImageUrl?: string;
  allImageUrls?: string[];
  structuredData: unknown[];
  pdfLinks: string[];
};

export type ScheduleItem = {
  id: string;
  capturedAt: string;
  room?: string;
  data: Record<string, string>;
  sourceUrl: string;
  sourceTitle: string;
  sourceImageUrl?: string;
  sourceImageUrls?: string[];
  sourcePdfLinks: string[];
};

export type Project = {
  id: string;
  name: string;
  address: string;
  description: string;
  updatedAt: string;
  isOwned: boolean;
};

export type Schedule = {
  id: string;
  projectId: string;
  name: string;
  kind: string;
  notes: string;
  position: number;
};

export type ScheduleColumn = {
  id: string;
  scheduleId: string;
  key: string;
  label: string;
  kind: string;
  position: number;
};

export type ActiveContext = {
  projectId: string;
  scheduleId: string;
};

export type ColumnDefinition = {
  key: string;
  label: string;
};

export type ExtractSpecRequest = {
  requestId: string;
  sentAt: string;
  client: ClientInfo;
  page: CapturedPage;
  projectContext: ProjectContext;
  scheduleId?: string;
  customColumns?: ColumnDefinition[];
  options: ExtractSpecOptions;
};

export type ClientInfo = {
  app: string;
  version: string;
  chromeVersion: string;
};

export type ProjectContext = {
  projectName: string;
  knownCategories: string[];
  knownRooms: string[];
  knownScheduleNames: string[];
};

export type ExtractSpecOptions = {
  includeDebug: boolean;
  returnAlternatives: boolean;
};

export type ExtractSpecResponse = {
  requestId: string;
  status: "ok" | "error";
  proposal?: ExtractedProposal;
  analysis?: ExtractAnalysis;
  error?: ExtractErrorPayload;
  meta: ExtractResponseMeta;
  nextCode?: string;
  knownRooms?: string[];
};

export type ExtractedProposal = {
  title: string;
  manufacturer: string;
  modelNumber: string;
  category: string;
  description: string;
  finish: string;
  finishModelNumber?: string;
  availableFinishes: string[];
  finishModelMappings: FinishModelMapping[];
  requiredAddOns: string[];
  optionalCompanions: string[];
  room?: string;
  suggestedScheduleName?: string;
  price?: string;
  leadTime?: string;
  stockStatus?: string;
  stockCount?: string;
  sourceUrl: string;
  sourceTitle: string;
  sourceImageUrl?: string;
  sourceImageUrls?: string[];
  sourcePdfLinks: string[];
  customFields?: Record<string, string>;
};

export type FinishModelMapping = {
  finish: string;
  modelNumber: string;
};

export type ExtractAnalysis = {
  missingFields: string[];
  warnings: string[];
  confidence: ExtractConfidence;
};

export type ExtractConfidence = {
  overall: number;
  title: number;
  manufacturer: number;
  modelNumber: number;
  category: number;
  description: number;
  finish: number;
  requiredAddOns: number;
};

export type ExtractErrorPayload = {
  code: string;
  message: string;
};

export type ExtractResponseMeta = {
  provider: string;
  model: string;
  promptVersion: string;
  durationMs: number;
};
