const year=document.getElementById("year");if(year)year.textContent=new Date().getFullYear();
const serviceNames={
  "deska-cash":"DesKaCash — E-Wallet",
  "deska-pay":"DesKaPay — Merchant & Payment",
  indochain:"IndoChain — Blockchain",
  indoscan:"IndoScan — Explorer",
  "indochain-wallet":"IndoChainWallet — Blockchain Wallet",
  partnership:"Partnership",
  promotor:"Promotor / Business Development",
  investor:"Investor / Strategic Partner",
  integration:"Integration / Technical Collaboration"
};
const serviceSelect=document.getElementById("service"),serviceSummary=document.getElementById("serviceSummary"),orderForm=document.getElementById("orderForm"),success=document.getElementById("success");
if(serviceSelect){
  const params=new URLSearchParams(location.search),selected=params.get("service");
  if(selected&&serviceNames[selected])serviceSelect.value=selected;
  const update=()=>{if(serviceSummary)serviceSummary.textContent=serviceNames[serviceSelect.value]||"Pilih topik"};
  serviceSelect.addEventListener("change",update);update();
}
if(orderForm){
  orderForm.addEventListener("submit",e=>{
    e.preventDefault();
    const data=new FormData(orderForm),service=serviceNames[data.get("service")]||data.get("service");
    document.getElementById("successText").textContent="Pengajuan tercatat untuk "+service+". Tim akan menindaklanjuti sesuai status produk dan jenis hubungan yang diajukan.";
    success.classList.add("show");orderForm.reset();
    if(serviceSelect){serviceSelect.value="deska-cash";serviceSelect.dispatchEvent(new Event("change"));}
  });
}