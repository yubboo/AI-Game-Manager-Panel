import type { JsonObject, ToolAuthorization, ToolObservation, ToolSpec } from '../../protocol/src/index.ts'
export interface ToolContext { runId:string; signal:AbortSignal; authorization?:ToolAuthorization; emitProgress(summary:string,data?:unknown):void }
export interface Tool { readonly spec:ToolSpec; execute(argumentsValue:JsonObject,context:ToolContext):Promise<{content:string;data?:unknown}> }
export interface ApprovalGrant { approved:boolean; requestHash:string }
export interface ApprovalController { approve(spec:ToolSpec,args:JsonObject,runId:string,signal:AbortSignal):Promise<ApprovalGrant> }
export class AllowReadOnlyApproval implements ApprovalController { async approve(spec:ToolSpec):Promise<ApprovalGrant>{return{approved:spec.risk==='read',requestHash:''}} }
export class ToolRegistry {
  #tools=new Map<string,Tool>(); constructor(private readonly approval:ApprovalController){}
  register(tool:Tool){if(this.#tools.has(tool.spec.name))throw new Error(`duplicate tool: ${tool.spec.name}`);this.#tools.set(tool.spec.name,tool)}
  specs(){return[...this.#tools.values()].map(t=>structuredClone(t.spec))}
  async execute(name:string,args:JsonObject,context:ToolContext):Promise<ToolObservation>{const started=Date.now();const tool=this.#tools.get(name);if(!tool)return{callId:'',name,ok:false,content:`Unknown tool: ${name}`,durationMs:Date.now()-started};let authorization:ToolAuthorization|undefined;if(tool.spec.risk!=='read'){const grant=await this.approval.approve(tool.spec,args,context.runId,context.signal);if(!grant.approved)return{callId:'',name,ok:false,content:'User denied this tool call.',durationMs:Date.now()-started};authorization={requestHash:grant.requestHash}}try{const result=await tool.execute(args,authorization?{...context,authorization}:context);return{callId:'',name,ok:true,content:result.content,...(result.data===undefined?{}:{data:result.data}),durationMs:Date.now()-started}}catch(error){return{callId:'',name,ok:false,content:error instanceof Error?error.message:String(error),durationMs:Date.now()-started}}}
}
