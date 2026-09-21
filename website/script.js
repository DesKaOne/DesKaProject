const year=document.getElementById("year");if(year)year.textContent=new Date().getFullYear();
const serviceNames={
  "deska-cash":"DesKaCash — Digital Wallet",
  "deska-pay":"DesKaPay — Payment Layer",
  indochain:"IndoChain — Blockchain Infrastructure",
  indoscan:"IndoScan — Explorer",
  "indochain-wallet":"IndoChainWallet — Blockchain Wallet",
  integration:"Integrasi Digital / API / Backend"
};
const serviceSelect=document.getElementById("service"),serviceSummary=document.getElementById("serviceSummary"),orderForm=document.getElementById("orderForm"),success=document.getElementById("success");
if(serviceSelect){
  const params=new URLSearchParams(location.search),selected=params.get("service");
  if(selected&&serviceNames[selected])serviceSelect.value=selected;
  const update=()=>{if(serviceSummary)serviceSummary.textContent=serviceNames[serviceSelect.value]||"Pilih produk/kebutuhan"};
  serviceSelect.addEventListener("change",update);update();
}
if(orderForm){
  orderForm.addEventListener("submit",e=>{
    e.preventDefault();
    const data=new FormData(orderForm),service=serviceNames[data.get("service")]||data.get("service");
    document.getElementById("successText").textContent="Pengajuan tercatat untuk "+service+". Tim akan mengonfirmasi status produk, kebutuhan, dan jika relevan detail komersial sebelum transaksi.";
    success.classList.add("show");
    orderForm.reset();
    if(serviceSelect){
      const params=new URLSearchParams(location.search);
      serviceSelect.value=params.get("service")||"deska-cash";
      serviceSelect.dispatchEvent(new Event("change"));
    }
  });
}