"use client";
import { RefreshCw } from "lucide-react";
export default function ErrorPage({reset}:{error:Error;reset:()=>void}){return <div className="error-box">No se pudo cargar la información del servidor. <button className="button" onClick={reset} style={{marginLeft:12}}><RefreshCw size={14}/>Reintentar</button></div>}
