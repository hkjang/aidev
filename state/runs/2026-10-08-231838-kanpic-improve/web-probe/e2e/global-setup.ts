/** 이 실행이 언제 시작했는지 적어 두면, 정리가 이 실행이 만든 것만 지운다. */
export default async function globalSetup(){
  process.env.KANPIC_E2E_STARTED_AT=new Date().toISOString()
}
