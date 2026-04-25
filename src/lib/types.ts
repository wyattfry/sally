export type CapturedPage = {
  title: string;
  url: string;
  visibleText: string;
  mainImageUrl?: string;
  structuredData: unknown[];
  pdfLinks: string[];
};

export type ScheduleItem = {
  id: string;
  capturedAt: string;
  zone: string;
  title: string;
  manufacturer: string;
  modelNumber: string;
  category: string;
  description: string;
  finish: string;
  finishModelNumber?: string;
  requiredAddOns: string[];
  optionalCompanions: string[];
  sourceUrl: string;
  sourceTitle: string;
  sourceImageUrl?: string;
  sourcePdfLinks: string[];
};

export type ExtractSpecRequest = {
  requestId: string;
  sentAt: string;
  client: ClientInfo;
  page: CapturedPage;
  projectContext: ProjectContext;
  options: ExtractSpecOptions;
};

export type ClientInfo = {
  app: string;
  version: string;
  chromeVersion: string;
};

export type ProjectContext = {
  projectName: string;
  knownZones: string[];
  knownCategories: string[];
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
  zone: string;
  sourceUrl: string;
  sourceTitle: string;
  sourceImageUrl?: string;
  sourcePdfLinks: string[];
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
