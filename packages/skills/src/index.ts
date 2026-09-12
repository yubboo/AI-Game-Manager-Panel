import { readFile } from 'node:fs/promises'
export interface SkillDefinition{id:string;description:string;file:string}
export class SkillRegistry{#skills=new Map<string,SkillDefinition>();register(skill:SkillDefinition){this.#skills.set(skill.id,skill)}list(){return[...this.#skills.values()].map(v=>({...v}))}async load(id:string){const skill=this.#skills.get(id);if(!skill)throw new Error(`Unknown skill: ${id}`);return readFile(skill.file,'utf8')}}
